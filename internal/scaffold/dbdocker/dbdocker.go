// Package dbdocker manages the database containers the scaffolder can start,
// and writes the docker-compose file the generated project ships with.
//
// It depends only on config and command, so it is a leaf of the scaffolding
// dependency graph and can be tested without a terminal.
//
// PostgreSQL and MySQL used to be two near-identical 60-line functions plus two
// near-identical compose templates. They are now one engine descriptor driving
// shared code; adding a third backend means adding a descriptor.
package dbdocker

import (
	"context"
	"crypto/rand"
	"encoding/hex"
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
	// commandTimeout bounds short control-plane calls (version, info, ps, stop,
	// rm). A hung Docker daemon must not hang the caller with no way out.
	commandTimeout = 30 * time.Second

	// pullTimeout bounds image pulls and container starts, which are
	// network-bound.
	pullTimeout = 5 * time.Minute

	// passwordLength is the length in hex characters of generated passwords.
	passwordLength = 16
)

// listenTCP is swappable so port discovery can be tested without binding.
var listenTCP = net.Listen

// Reporter receives human-readable progress.
//
// It is declared here rather than imported so dbdocker stays a leaf package;
// scaffold.Env satisfies it structurally.
type Reporter interface {
	Step(message string)
}

// Credentials describe a container this package started. The caller stores
// these so the generated .env matches the running container — writing .env
// before the container exists produces a password mismatch.
type Credentials struct {
	Port          string
	Password      string
	RootPassword  string
	DatabaseName  string
	ConnectionURL string
}

// engine describes one supported database backend.
type engine struct {
	Name         string
	Image        string
	InternalPort int
	DefaultPort  int
	// ContainerPrefix distinguishes containers per engine.
	ContainerPrefix string
	// Env builds the container environment for the generated credentials.
	Env func(c Credentials) []string
	// URL renders the connection string.
	URL func(c Credentials) string
	// ReadyDelay is how long to wait for the server to accept connections.
	ReadyDelay time.Duration
	// NeedsRootPassword requests a second generated secret.
	NeedsRootPassword bool
}

var postgres = engine{
	Name:            "PostgreSQL",
	Image:           "postgres:16",
	InternalPort:    5432,
	DefaultPort:     5432,
	ContainerPrefix: "django-stack",
	Env: func(c Credentials) []string {
		return []string{
			"POSTGRES_PASSWORD=" + c.Password,
			"POSTGRES_DB=" + c.DatabaseName,
		}
	},
	URL: func(c Credentials) string {
		return fmt.Sprintf("postgresql://postgres:%s@localhost:%s/%s", c.Password, c.Port, c.DatabaseName)
	},
	ReadyDelay: 5 * time.Second,
}

var mysql = engine{
	Name:            "MySQL",
	Image:           "mysql:8.0",
	InternalPort:    3306,
	DefaultPort:     3306,
	ContainerPrefix: "django-stack-mysql",
	Env: func(c Credentials) []string {
		return []string{
			"MYSQL_ROOT_PASSWORD=" + c.RootPassword,
			"MYSQL_DATABASE=" + c.DatabaseName,
			"MYSQL_USER=django_user",
			"MYSQL_PASSWORD=" + c.Password,
		}
	},
	URL: func(c Credentials) string {
		return fmt.Sprintf("mysql://django_user:%s@localhost:%s/%s", c.Password, c.Port, c.DatabaseName)
	},
	ReadyDelay:        10 * time.Second,
	NeedsRootPassword: true,
}

// Available reports whether the docker CLI is installed.
func Available(ctx context.Context, run command.Runner) bool {
	ctx, cancel := context.WithTimeout(ctx, commandTimeout)
	defer cancel()

	_, err := run.Run(ctx, "", "docker", "--version")
	return err == nil
}

// Running reports whether the docker daemon is reachable.
func Running(ctx context.Context, run command.Runner) bool {
	ctx, cancel := context.WithTimeout(ctx, commandTimeout)
	defer cancel()

	_, err := run.Run(ctx, "", "docker", "info")
	return err == nil
}

// SetupPostgres starts a PostgreSQL container for the project.
func SetupPostgres(ctx context.Context, run command.Runner, rep Reporter, projectName string) (Credentials, error) {
	return setup(ctx, run, rep, postgres, projectName)
}

// SetupMySQL starts a MySQL container for the project.
func SetupMySQL(ctx context.Context, run command.Runner, rep Reporter, projectName string) (Credentials, error) {
	return setup(ctx, run, rep, mysql, projectName)
}

func setup(ctx context.Context, run command.Runner, rep Reporter, e engine, projectName string) (Credentials, error) {
	if !Available(ctx, run) {
		return Credentials{}, fmt.Errorf("docker is not installed; install it from https://docker.com")
	}
	if !Running(ctx, run) {
		return Credentials{}, fmt.Errorf("docker is not running; start Docker Desktop and try again")
	}

	step(rep, "Checking for "+e.Image+" image...")
	if err := pullImage(ctx, run, e); err != nil {
		// A failed pull is not fatal: the image may already be cached locally.
		step(rep, "Warning: could not pull "+e.Image+", trying an existing local image...")
	}

	creds := Credentials{DatabaseName: DatabaseName(projectName)}

	var err error
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

	containerName := fmt.Sprintf("%s-%s-%s", e.ContainerPrefix, projectName, creds.Port)
	step(rep, fmt.Sprintf("Starting %s container on port %s...", e.Name, creds.Port))

	args := []string{"run", "--name", containerName}
	for _, kv := range e.Env(creds) {
		args = append(args, "-e", kv)
	}
	args = append(args,
		"-p", fmt.Sprintf("%s:%d", creds.Port, e.InternalPort),
		"-d", e.Image,
	)

	runCtx, cancel := context.WithTimeout(ctx, pullTimeout)
	defer cancel()

	if _, err := run.Run(runCtx, "", "docker", args...); err != nil {
		return Credentials{}, fmt.Errorf("failed to start %s container: %w", e.Name, err)
	}

	step(rep, e.Name+" container started. Waiting for it to be ready...")
	sleep(ctx, e.ReadyDelay)

	creds.ConnectionURL = e.URL(creds)

	step(rep, e.Name+" ready.")
	step(rep, "Connection: localhost:"+creds.Port)
	step(rep, "Database: "+creds.DatabaseName)
	step(rep, "Password: "+creds.Password+" (save this!)")

	return creds, nil
}

// Stop stops and removes the containers this project started.
//
// Failures are reported rather than swallowed: a container left holding the
// database port makes the next run fail with an unrelated bind error.
func Stop(ctx context.Context, run command.Runner, projectName string) error {
	ctx, cancel := context.WithTimeout(ctx, commandTimeout)
	defer cancel()

	filter := fmt.Sprintf("name=django-stack-%s-*", projectName)
	output, err := run.Run(ctx, "", "docker", "ps", "-q", "--filter", filter)
	if err != nil {
		return fmt.Errorf("failed to list containers for %s: %w", projectName, err)
	}

	containers := strings.TrimSpace(string(output))
	if containers == "" {
		return nil
	}

	if _, err := run.Run(ctx, "", "docker", "stop", containers); err != nil {
		return fmt.Errorf("failed to stop containers %s: %w", containers, err)
	}
	if _, err := run.Run(ctx, "", "docker", "rm", containers); err != nil {
		return fmt.Errorf("failed to remove containers %s: %w", containers, err)
	}
	return nil
}

func pullImage(ctx context.Context, run command.Runner, e engine) error {
	ctx, cancel := context.WithTimeout(ctx, pullTimeout)
	defer cancel()

	_, err := run.Run(ctx, "", "docker", "pull", e.Image)
	return err
}

// DatabaseName derives a legal database name from the project name.
func DatabaseName(projectName string) string {
	base := filepath.Base(filepath.Clean(projectName))
	return strings.ToLower(strings.ReplaceAll(base, "-", "_"))
}

// Password returns a random hex password of the given length.
func Password(length int) (string, error) {
	// Two hex characters per byte, so half as many bytes are needed.
	buf := make([]byte, (length+1)/2)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf)[:length], nil
}

// AvailablePort finds a free TCP port at or after preferred.
//
// This is inherently racy — the port is released before Docker binds it — so
// callers must be prepared for the bind to fail and retry.
func AvailablePort(preferred int) (string, error) {
	for port := preferred; port < preferred+100; port++ {
		listener, err := listenTCP("tcp", fmt.Sprintf("127.0.0.1:%d", port))
		if err != nil {
			continue
		}
		_ = listener.Close()
		return strconv.Itoa(port), nil
	}

	listener, err := listenTCP("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}
	defer func() { _ = listener.Close() }()

	addr, ok := listener.Addr().(*net.TCPAddr)
	if !ok || addr.Port == 0 {
		return "", fmt.Errorf("failed to determine a free TCP port")
	}
	return strconv.Itoa(addr.Port), nil
}

// WriteComposeFile writes docker-compose.yml for the given database into dir.
func WriteComposeFile(dir string, db config.Database) error {
	var content string
	switch db {
	case config.PostgresDocker:
		content = composeFile(postgres, "postgres_data:/var/lib/postgresql/data", "postgres_data",
			`["CMD-SHELL", "pg_isready -U ${DB_USER}"]`, "")
	case config.MySQLDocker:
		content = composeFile(mysql, "mysql_data:/var/lib/mysql", "mysql_data",
			`["CMD", "mysqladmin", "ping", "-h", "localhost"]`,
			"    command: --default-authentication-plugin=mysql_native_password\n")
	default:
		return fmt.Errorf("no docker-compose file for database %q", db)
	}

	return os.WriteFile(filepath.Join(dir, "docker-compose.yml"), []byte(content), 0o644)
}

func composeFile(e engine, volumeMount, volumeName, healthcheck, extra string) string {
	var env strings.Builder
	switch e.Name {
	case "PostgreSQL":
		env.WriteString("      POSTGRES_DB: ${DB_NAME}\n")
		env.WriteString("      POSTGRES_USER: ${DB_USER}\n")
		env.WriteString("      POSTGRES_PASSWORD: ${DB_PASSWORD}\n")
	case "MySQL":
		env.WriteString("      MYSQL_DATABASE: ${DB_NAME}\n")
		env.WriteString("      MYSQL_USER: ${DB_USER}\n")
		env.WriteString("      MYSQL_PASSWORD: ${DB_PASSWORD}\n")
		env.WriteString("      MYSQL_ROOT_PASSWORD: ${DB_ROOT_PASSWORD}\n")
	}

	return fmt.Sprintf(`services:
  db:
    image: %s
    restart: unless-stopped
    env_file:
      - .env
    environment:
%s    ports:
      - "${DB_PORT}:%d"
    volumes:
      - %s
%s    healthcheck:
      test: %s
      interval: 10s
      timeout: 5s
      retries: 5

volumes:
  %s:
`, e.Image, env.String(), e.InternalPort, volumeMount, extra, healthcheck, volumeName)
}

func step(rep Reporter, message string) {
	if rep != nil {
		rep.Step(message)
	}
}

// sleep waits for d, or returns early if ctx is cancelled. It is a variable so
// tests can skip the container warm-up wait.
var sleep = func(ctx context.Context, d time.Duration) {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-timer.C:
	case <-ctx.Done():
	}
}
