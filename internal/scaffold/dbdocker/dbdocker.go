package dbdocker

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/vaibdix/django-cli/internal/config"
	"github.com/vaibdix/django-cli/internal/scaffold/command"
)

const (
	commandTimeout = 30 * time.Second

	pullTimeout = 5 * time.Minute

	readyTimeout = 3 * time.Minute

	passwordLength = 16

	loopbackInterface = "127.0.0.1"

	ComposeFile = "docker-compose.yml"

	Service = "db"
)

var listenTCP = net.Listen

type Reporter interface {
	Step(message string)
}

type Credentials struct {
	Port         string
	Password     string
	RootPassword string
	DatabaseName string
}

type engine struct {
	Name         string
	Image        string
	InternalPort int
	DefaultPort  int
	Volume       string
	DataDir      string
	Environment  []string
	Healthcheck  string

	NeedsRootPassword bool
}

var postgres = engine{
	Name:         "PostgreSQL",
	Image:        "postgres:18",
	InternalPort: 5432,
	DefaultPort:  5432,
	Volume:       "postgres_data",
	DataDir:      "/var/lib/postgresql",
	Environment: []string{
		"POSTGRES_DB: ${DB_NAME:?set DB_NAME in .env}",
		"POSTGRES_USER: ${DB_USER:?set DB_USER in .env}",
		"POSTGRES_PASSWORD: ${DB_PASSWORD:?set DB_PASSWORD in .env}",
	},
	Healthcheck: `["CMD-SHELL", "pg_isready -U ${DB_USER} -d ${DB_NAME}"]`,
}

var mysql = engine{
	Name:         "MySQL",
	Image:        "mysql:8.4",
	InternalPort: 3306,
	DefaultPort:  3306,
	Volume:       "mysql_data",
	DataDir:      "/var/lib/mysql",
	Environment: []string{
		"MYSQL_DATABASE: ${DB_NAME:?set DB_NAME in .env}",
		"MYSQL_USER: ${DB_USER:?set DB_USER in .env}",
		"MYSQL_PASSWORD: ${DB_PASSWORD:?set DB_PASSWORD in .env}",
		"MYSQL_ROOT_PASSWORD: ${DB_ROOT_PASSWORD:?set DB_ROOT_PASSWORD in .env}",
	},
	Healthcheck:       `["CMD-SHELL", "mysqladmin ping -h localhost -u ${DB_USER} -p${DB_PASSWORD}"]`,
	NeedsRootPassword: true,
}

func engineFor(db config.Database) (engine, error) {
	switch db {
	case config.PostgresDocker:
		return postgres, nil
	case config.MySQLDocker:
		return mysql, nil
	default:
		return engine{}, fmt.Errorf("no docker database for %q", db)
	}
}

func Available(ctx context.Context, run command.Runner) bool {
	ctx, cancel := context.WithTimeout(ctx, commandTimeout)
	defer cancel()

	_, err := run.Run(ctx, "", "docker", "--version")
	return err == nil
}

func ComposeAvailable(ctx context.Context, run command.Runner) bool {
	ctx, cancel := context.WithTimeout(ctx, commandTimeout)
	defer cancel()

	_, err := run.Run(ctx, "", "docker", "compose", "version")
	return err == nil
}

func Running(ctx context.Context, run command.Runner) bool {
	ctx, cancel := context.WithTimeout(ctx, commandTimeout)
	defer cancel()

	_, err := run.Run(ctx, "", "docker", "info")
	return err == nil
}

func NewCredentials(db config.Database, projectName string) (Credentials, error) {
	e, err := engineFor(db)
	if err != nil {
		return Credentials{}, err
	}

	creds := Credentials{DatabaseName: DatabaseName(projectName)}
	if creds.Password, err = Password(passwordLength); err != nil {
		return Credentials{}, fmt.Errorf("failed to generate a database password: %w", err)
	}
	if e.NeedsRootPassword {
		if creds.RootPassword, err = Password(passwordLength); err != nil {
			return Credentials{}, fmt.Errorf("failed to generate a root password: %w", err)
		}
	}
	if creds.Port, err = AvailablePort(e.DefaultPort); err != nil {
		return Credentials{}, fmt.Errorf("failed to find a free port for %s: %w", e.Name, err)
	}
	return creds, nil
}

func ProjectName(projectName string) string {
	return DatabaseName(projectName)
}

func VolumeName(projectName string, db config.Database) string {
	e, err := engineFor(db)
	if err != nil {
		return ""
	}
	return ProjectName(projectName) + "_" + e.Volume
}

func Start(ctx context.Context, run command.Runner, rep Reporter, dir, projectName string, db config.Database) error {
	e, err := engineFor(db)
	if err != nil {
		return err
	}
	if !Available(ctx, run) {
		return errors.New("docker is not installed; install Docker Desktop or OrbStack from https://docker.com")
	}
	if !ComposeAvailable(ctx, run) {
		return errors.New("docker compose is not available; install Docker Compose v2 (bundled with Docker Desktop and OrbStack)")
	}
	if !Running(ctx, run) {
		return errors.New("docker is not running; start Docker Desktop or OrbStack and try again")
	}
	if err := refuseStaleVolume(ctx, run, VolumeName(projectName, db)); err != nil {
		return err
	}

	compose := func(ctx context.Context, args ...string) ([]byte, error) {
		return run.Run(ctx, dir, "docker", append([]string{"compose"}, args...)...)
	}

	step(rep, "Pulling "+e.Image+"...")
	pullCtx, cancelPull := context.WithTimeout(ctx, pullTimeout)
	defer cancelPull()
	if _, err := compose(pullCtx, "pull", Service); err != nil {
		step(rep, "Warning: could not pull "+e.Image+", trying an existing local image...")
	}

	step(rep, fmt.Sprintf("Starting %s with docker compose and waiting for it to be healthy...", e.Name))
	upCtx, cancelUp := context.WithTimeout(ctx, readyTimeout+pullTimeout)
	defer cancelUp()
	waitSeconds := strconv.Itoa(int(readyTimeout.Seconds()))
	if _, err := compose(upCtx, "up", "-d", "--wait", "--wait-timeout", waitSeconds, Service); err != nil {
		cleanupCtx, cancelCleanup := context.WithTimeout(context.WithoutCancel(ctx), commandTimeout)
		defer cancelCleanup()
		logs, _ := compose(cleanupCtx, "logs", "--tail", "30", Service)
		_, _ = compose(cleanupCtx, "down", "-v")
		return fmt.Errorf("%s did not become healthy: %w\n%s", e.Name, err, strings.TrimSpace(string(logs)))
	}

	step(rep, e.Name+" is healthy.")
	return nil
}

func Stop(ctx context.Context, run command.Runner, dir string) error {
	ctx, cancel := context.WithTimeout(ctx, commandTimeout)
	defer cancel()

	_, err := run.Run(ctx, dir, "docker", "compose", "down", "-v")
	return err
}

func refuseStaleVolume(ctx context.Context, run command.Runner, volume string) error {
	ctx, cancel := context.WithTimeout(ctx, commandTimeout)
	defer cancel()

	if _, err := run.Run(ctx, "", "docker", "volume", "inspect", volume); err != nil {
		return nil
	}
	return fmt.Errorf(
		"a docker volume named %s already exists.\n\n"+
			"It belongs to an earlier project with this name and holds a different password, "+
			"so the new project could not log in.\n"+
			"Remove it (this deletes that database) or choose another project name:\n"+
			"  docker volume rm %s", volume, volume)
}

func DatabaseName(projectName string) string {
	base := filepath.Base(filepath.Clean(projectName))
	return strings.ToLower(strings.ReplaceAll(base, "-", "_"))
}

func Password(length int) (string, error) {
	buf := make([]byte, (length+1)/2)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf)[:length], nil
}

var probedInterfaces = []string{"0.0.0.0", loopbackInterface}

func AvailablePort(preferred int) (string, error) {
	for port := preferred; preferred > 0 && port < preferred+100; port++ {
		if portIsFree(port) {
			return strconv.Itoa(port), nil
		}
	}

	for range 20 {
		port, err := ephemeralPort()
		if err != nil {
			return "", err
		}
		if portIsFree(port) {
			return strconv.Itoa(port), nil
		}
	}
	return "", fmt.Errorf("failed to find a TCP port free on every interface")
}

func portIsFree(port int) bool {
	for _, host := range probedInterfaces {
		listener, err := listenTCP("tcp", net.JoinHostPort(host, strconv.Itoa(port)))
		if err != nil {
			return false
		}
		_ = listener.Close()
	}
	return true
}

func ephemeralPort() (int, error) {
	listener, err := listenTCP("tcp", net.JoinHostPort(loopbackInterface, "0"))
	if err != nil {
		return 0, err
	}
	defer func() { _ = listener.Close() }()

	addr, ok := listener.Addr().(*net.TCPAddr)
	if !ok || addr.Port == 0 {
		return 0, fmt.Errorf("failed to determine a free TCP port")
	}
	return addr.Port, nil
}

func WriteComposeFile(dir, projectName string, db config.Database) error {
	e, err := engineFor(db)
	if err != nil {
		return fmt.Errorf("no docker-compose file for database %q", db)
	}
	return os.WriteFile(filepath.Join(dir, ComposeFile), []byte(composeFile(e, ProjectName(projectName))), 0o644)
}

func composeFile(e engine, name string) string {
	var env strings.Builder
	for _, line := range e.Environment {
		env.WriteString("      " + line + "\n")
	}

	return fmt.Sprintf(`name: %s

services:
  %s:
    image: %s
    restart: unless-stopped
    environment:
%s    ports:
      - "%s:${DB_PORT}:%d"
    volumes:
      - %s:%s
    healthcheck:
      test: %s
      interval: 10s
      timeout: 5s
      retries: 5

volumes:
  %s:
`, name, Service, e.Image, env.String(), loopbackInterface, e.InternalPort, e.Volume, e.DataDir, e.Healthcheck, e.Volume)
}

func step(rep Reporter, message string) {
	if rep != nil {
		rep.Step(message)
	}
}

func ImageFor(database config.Database) string {
	e, err := engineFor(database)
	if err != nil {
		return ""
	}
	return e.Image
}
