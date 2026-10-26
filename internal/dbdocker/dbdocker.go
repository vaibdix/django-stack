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

	"github.com/vaibdix/django-stack/internal/command"
	"github.com/vaibdix/django-stack/internal/config"
	"github.com/vaibdix/django-stack/internal/report"
	"github.com/vaibdix/django-stack/internal/version"
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

type listenFunc func(network, address string) (net.Listener, error)

type Credentials struct {
	Port         string
	Password     string
	RootPassword string
	DatabaseName string
}

type engine struct {
	config.DatabaseSpec
	Volume      string
	DataDir     string
	Environment []string
	Healthcheck string
	InitName    string
	InitSQL     string
}

var engines = map[config.Database]engine{
	config.PostgresDocker: {
		Volume:  "postgres_data",
		DataDir: "/var/lib/postgresql",
		Environment: []string{
			"POSTGRES_DB: ${DB_NAME:?set DB_NAME in .env}",
			"POSTGRES_USER: postgres",
			"POSTGRES_PASSWORD: ${DB_ROOT_PASSWORD:?set DB_ROOT_PASSWORD in .env}",
		},
		Healthcheck: `["CMD-SHELL", "pg_isready -h 127.0.0.1 -U postgres -d ${DB_NAME}"]`,
		InitName:    "create-app-role",
		InitSQL:     `CREATE ROLE "${DB_USER:?set DB_USER in .env}" LOGIN CREATEDB PASSWORD '${DB_PASSWORD:?set DB_PASSWORD in .env}'; ALTER DATABASE "${DB_NAME}" OWNER TO "${DB_USER}";`,
	},
	config.MySQLDocker: {
		Volume:  "mysql_data",
		DataDir: "/var/lib/mysql",
		Environment: []string{
			"MYSQL_DATABASE: ${DB_NAME:?set DB_NAME in .env}",
			"MYSQL_USER: ${DB_USER:?set DB_USER in .env}",
			"MYSQL_PASSWORD: ${DB_PASSWORD:?set DB_PASSWORD in .env}",
			"MYSQL_ROOT_PASSWORD: ${DB_ROOT_PASSWORD:?set DB_ROOT_PASSWORD in .env}",
		},
		Healthcheck: `["CMD-SHELL", "MYSQL_PWD=\"$$MYSQL_PASSWORD\" mysqladmin ping -h 127.0.0.1 -u \"$$MYSQL_USER\""]`,
		InitName:    "grant-test-database",
		InitSQL:     "GRANT ALL PRIVILEGES ON `test_${DB_NAME}`.* TO '${DB_USER}'@'%';",
	},
}

func engineFor(db config.Database) (engine, error) {
	e, ok := engines[db]
	if !ok {
		return engine{}, fmt.Errorf("no docker database for %q", db)
	}
	e.DatabaseSpec = db.Spec()
	return e, nil
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
	if creds.RootPassword, err = Password(passwordLength); err != nil {
		return Credentials{}, fmt.Errorf("failed to generate a root password: %w", err)
	}
	if creds.Port, err = AvailablePort(e.Port); err != nil {
		return Credentials{}, fmt.Errorf("failed to find a free port for %s: %w", e.Label, err)
	}
	return creds, nil
}

func VolumeName(projectName string, db config.Database) string {
	e, err := engineFor(db)
	if err != nil {
		return ""
	}
	return DatabaseName(projectName) + "_" + e.Volume
}

func Ready(ctx context.Context, run command.Runner, projectName string, db config.Database) error {
	if !Available(ctx, run) {
		return errors.New("docker is not installed; install Docker Desktop or OrbStack from https://docker.com")
	}
	if !ComposeAvailable(ctx, run) {
		return errors.New("docker compose is not available; install Docker Compose v2 (bundled with Docker Desktop and OrbStack)")
	}
	if !Running(ctx, run) {
		return errors.New("docker is not running; start Docker Desktop or OrbStack and try again")
	}
	if err := requireEngineAPI(ctx, run); err != nil {
		return err
	}
	return refuseStaleVolume(ctx, run, VolumeName(projectName, db))
}

func Start(ctx context.Context, run command.Runner, rep report.Reporter, dir, projectName string, db config.Database) error {
	e, err := engineFor(db)
	if err != nil {
		return err
	}
	if err := Ready(ctx, run, projectName, db); err != nil {
		return err
	}

	compose := func(ctx context.Context, args ...string) ([]byte, error) {
		return run.Run(ctx, dir, "docker", append([]string{"compose"}, args...)...)
	}

	rep.Step("Pulling " + e.Image + "...")
	pullCtx, cancelPull := context.WithTimeout(ctx, pullTimeout)
	defer cancelPull()
	if output, err := compose(pullCtx, "pull", Service); err != nil {
		return fmt.Errorf("failed to pull %s: %w\n%s\nOnce the registry is reachable, run 'docker pull %s' and try again",
			e.Image, err, strings.TrimSpace(string(output)), e.Image)
	}

	rep.Step(fmt.Sprintf("Starting %s with docker compose and waiting for it to be healthy...", e.Label))
	upCtx, cancelUp := context.WithTimeout(ctx, readyTimeout+pullTimeout)
	defer cancelUp()
	waitSeconds := strconv.Itoa(int(readyTimeout.Seconds()))
	if _, err := compose(upCtx, "up", "-d", "--wait", "--wait-timeout", waitSeconds, Service); err != nil {
		cleanupCtx, cancelCleanup := context.WithTimeout(context.WithoutCancel(ctx), commandTimeout)
		defer cancelCleanup()
		logs, _ := compose(cleanupCtx, "logs", "--tail", "30", Service)
		_, _ = compose(cleanupCtx, "down", "-v")
		return fmt.Errorf("%s did not become healthy: %w\n%s", e.Label, err, strings.TrimSpace(string(logs)))
	}

	rep.Step(e.Label + " is healthy.")
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

	output, err := run.Run(ctx, "", "docker", "volume", "ls", "--filter", "name=^"+volume+"$", "--format", "{{.Name}}")
	if err != nil {
		return fmt.Errorf("failed to check for an existing %s volume: %w", volume, err)
	}
	if strings.TrimSpace(string(output)) == "" {
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

func AvailablePort(preferred int) (string, error) {
	return availablePort(net.Listen, preferred)
}

func availablePort(listen listenFunc, preferred int) (string, error) {
	for port := preferred; preferred > 0 && port < preferred+100; port++ {
		if portIsFree(listen, port) {
			return strconv.Itoa(port), nil
		}
	}

	for range 20 {
		port, err := ephemeralPort(listen)
		if err != nil {
			return "", err
		}
		if portIsFree(listen, port) {
			return strconv.Itoa(port), nil
		}
	}
	return "", errors.New("failed to find a TCP port free on every interface")
}

func portIsFree(listen listenFunc, port int) bool {
	for _, host := range []string{"0.0.0.0", loopbackInterface} {
		listener, err := listen("tcp", net.JoinHostPort(host, strconv.Itoa(port)))
		if err != nil {
			return false
		}
		_ = listener.Close()
	}
	return true
}

func ephemeralPort(listen listenFunc) (int, error) {
	listener, err := listen("tcp", net.JoinHostPort(loopbackInterface, "0"))
	if err != nil {
		return 0, err
	}
	defer func() { _ = listener.Close() }()

	addr, ok := listener.Addr().(*net.TCPAddr)
	if !ok || addr.Port == 0 {
		return 0, errors.New("failed to determine a free TCP port")
	}
	return addr.Port, nil
}

func WriteComposeFile(dir, projectName string, db config.Database) error {
	e, err := engineFor(db)
	if err != nil {
		return fmt.Errorf("no docker-compose file for database %q", db)
	}
	return os.WriteFile(filepath.Join(dir, ComposeFile), []byte(composeFile(e, DatabaseName(projectName))), 0o644)
}

func composeFile(e engine, name string) string {
	var env strings.Builder
	for _, line := range e.Environment {
		env.WriteString("      " + line + "\n")
	}

	var serviceConfigs, topLevelConfigs string
	if e.InitSQL != "" {
		serviceConfigs = fmt.Sprintf(`    configs:
      - source: %s
        target: /docker-entrypoint-initdb.d/10-%s.sql
`, e.InitName, e.InitName)
		topLevelConfigs = fmt.Sprintf(`
configs:
  %s:
    content: |
      %s
`, e.InitName, e.InitSQL)
	}

	return fmt.Sprintf(`name: %s

services:
  %s:
    image: %s
    restart: unless-stopped
    environment:
%s    ports:
      - "%s:${DB_PORT:?set DB_PORT in .env}:%d"
    volumes:
      - %s:%s
%s    healthcheck:
      test: %s
      interval: 10s
      timeout: 5s
      retries: 5
      start_period: 60s
      start_interval: 1s

volumes:
  %s:
%s`, name, Service, e.Image, env.String(), loopbackInterface, e.Port, e.Volume, e.DataDir,
		serviceConfigs, e.Healthcheck, e.Volume, topLevelConfigs)
}

const minimumEngineAPIText = "1.44"

var minimumEngineAPI = []int{1, 44}

func requireEngineAPI(ctx context.Context, run command.Runner) error {
	ctx, cancel := context.WithTimeout(ctx, commandTimeout)
	defer cancel()

	output, err := run.Run(ctx, "", "docker", "version", "--format", "{{.Server.APIVersion}}")
	if err != nil {
		return fmt.Errorf("failed to read the docker engine API version: %w", err)
	}
	reported := strings.TrimSpace(string(output))
	if reported == "" {
		return errors.New("docker version reported no engine API version, so the engine cannot be checked against API " + minimumEngineAPIText)
	}
	api, err := version.Parse(reported)
	if err != nil {
		return fmt.Errorf("docker reported an engine API version django-stack cannot read: %w", err)
	}
	if version.Compare(api, minimumEngineAPI) >= 0 {
		return nil
	}
	return fmt.Errorf("docker engine API %s is too old: the generated compose file needs Docker Engine 25 "+
		"(API %s) or newer for its startup healthcheck; update Docker Desktop or OrbStack", reported, minimumEngineAPIText)
}
