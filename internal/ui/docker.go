package ui

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

var listenTCP = net.Listen

func isDockerAvailable() bool {
	cmd := exec.Command("docker", "--version")
	return cmd.Run() == nil
}

func isDockerRunning() bool {
	cmd := exec.Command("docker", "info")
	return cmd.Run() == nil
}

func pullPostgresImage() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", "pull", "postgres:16")
	cmd.Stdout = nil
	cmd.Stderr = nil

	return cmd.Run()
}

func generatePassword(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes)[:length], nil
}

func getAvailablePort(preferred int) (string, error) {
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
	defer listener.Close()

	addr, ok := listener.Addr().(*net.TCPAddr)
	if !ok || addr.Port == 0 {
		return "", fmt.Errorf("failed to determine a free TCP port")
	}
	return strconv.Itoa(addr.Port), nil
}

func (m *Model) SetupDockerPostgres(projectPath string) (string, error) {
	if !isDockerAvailable() {
		return "", fmt.Errorf("Docker is not installed. Please install Docker from https://docker.com")
	}

	if !isDockerRunning() {
		return "", fmt.Errorf("Docker is not running. Please start Docker Desktop")
	}

	m.AppendStepMessage("Checking for postgres:16 image...")
	if err := pullPostgresImage(); err != nil {
		m.AppendStepMessage("Warning: Could not pull postgres:16 image, trying to use existing...")
	}

	password, err := generatePassword(16)
	if err != nil {
		password = "postgres"
	}

	port, err := getAvailablePort(5432)
	if err != nil || port == "" {
		port = "5432"
	}

	dbName := strings.ToLower(strings.ReplaceAll(m.ProjectName, "-", "_"))

	containerName := fmt.Sprintf("django-stack-%s-%s", m.ProjectName, port)

	m.AppendStepMessage(fmt.Sprintf("Starting PostgreSQL container on port %s...", port))

	cmd := exec.Command(
		"docker", "run",
		"--name", containerName,
		"-e", fmt.Sprintf("POSTGRES_PASSWORD=%s", password),
		"-e", fmt.Sprintf("POSTGRES_DB=%s", dbName),
		"-p", fmt.Sprintf("%s:5432", port),
		"-d",
		"postgres:16",
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to start PostgreSQL container: %w\nOutput: %s", err, string(output))
	}

	m.AppendStepMessage("PostgreSQL container started. Waiting for it to be ready...")

	time.Sleep(5 * time.Second)

	connStr := fmt.Sprintf("postgresql://postgres:%s@localhost:%s/%s", password, port, dbName)

	m.PostgresDockerPort = port
	m.PostgresDockerPassword = password

	m.AppendStepMessage(fmt.Sprintf("PostgreSQL ready!"))
	m.AppendStepMessage(fmt.Sprintf("Connection: localhost:%s", port))
	m.AppendStepMessage(fmt.Sprintf("Database: %s", dbName))
	m.AppendStepMessage(fmt.Sprintf("User: postgres"))
	m.AppendStepMessage(fmt.Sprintf("Password: %s (save this!)", password))

	return connStr, nil
}

func generateDockerComposeFile(projectPath string) error {
	content := `services:
  db:
    image: postgres:16
    restart: unless-stopped
    env_file:
      - .env
    environment:
      POSTGRES_DB: ${DB_NAME}
      POSTGRES_USER: ${DB_USER}
      POSTGRES_PASSWORD: ${DB_PASSWORD}
    ports:
      - "${DB_PORT}:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ${DB_USER}"]
      interval: 10s
      timeout: 5s
      retries: 5

volumes:
  postgres_data:
`
	composePath := filepath.Join(projectPath, "docker-compose.yml")
	return os.WriteFile(composePath, []byte(content), 0644)
}

func pullMySQLImage() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", "pull", "mysql:8.0")
	cmd.Stdout = nil
	cmd.Stderr = nil

	return cmd.Run()
}

func (m *Model) SetupDockerMySQL(projectPath string) (string, error) {
	if !isDockerAvailable() {
		return "", fmt.Errorf("Docker is not installed. Please install Docker from https://docker.com")
	}

	if !isDockerRunning() {
		return "", fmt.Errorf("Docker is not running. Please start Docker Desktop")
	}

	m.AppendStepMessage("Checking for mysql:8.0 image...")
	if err := pullMySQLImage(); err != nil {
		m.AppendStepMessage("Warning: Could not pull mysql:8.0 image, trying to use existing...")
	}

	password, err := generatePassword(16)
	if err != nil {
		password = "mysql"
	}

	rootPassword, err := generatePassword(16)
	if err != nil {
		rootPassword = "rootpassword"
	}

	port, err := getAvailablePort(3306)
	if err != nil || port == "" {
		port = "3306"
	}

	dbName := strings.ToLower(strings.ReplaceAll(m.ProjectName, "-", "_"))

	containerName := fmt.Sprintf("django-stack-mysql-%s-%s", m.ProjectName, port)

	m.AppendStepMessage(fmt.Sprintf("Starting MySQL container on port %s...", port))

	cmd := exec.Command(
		"docker", "run",
		"--name", containerName,
		"-e", fmt.Sprintf("MYSQL_ROOT_PASSWORD=%s", rootPassword),
		"-e", fmt.Sprintf("MYSQL_DATABASE=%s", dbName),
		"-e", "MYSQL_USER=django_user",
		"-e", fmt.Sprintf("MYSQL_PASSWORD=%s", password),
		"-p", fmt.Sprintf("%s:3306", port),
		"-d",
		"mysql:8.0",
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to start MySQL container: %w\nOutput: %s", err, string(output))
	}

	m.AppendStepMessage("MySQL container started. Waiting for it to be ready...")

	time.Sleep(10 * time.Second)

	connStr := fmt.Sprintf("mysql://django_user:%s@localhost:%s/%s", password, port, dbName)

	m.MySQLDockerPort = port
	m.MySQLDockerPassword = password

	m.AppendStepMessage("MySQL ready!")
	m.AppendStepMessage(fmt.Sprintf("Connection: localhost:%s", port))
	m.AppendStepMessage(fmt.Sprintf("Database: %s", dbName))
	m.AppendStepMessage("User: django_user")
	m.AppendStepMessage(fmt.Sprintf("Password: %s (save this!)", password))

	return connStr, nil
}

func generateDockerComposeFileMySQL(projectPath string) error {
	content := `services:
  db:
    image: mysql:8.0
    restart: unless-stopped
    env_file:
      - .env
    environment:
      MYSQL_DATABASE: ${DB_NAME}
      MYSQL_USER: ${DB_USER}
      MYSQL_PASSWORD: ${DB_PASSWORD}
      MYSQL_ROOT_PASSWORD: ${DB_ROOT_PASSWORD}
    ports:
      - "${DB_PORT}:3306"
    volumes:
      - mysql_data:/var/lib/mysql
    command: --default-authentication-plugin=mysql_native_password
    healthcheck:
      test: ["CMD", "mysqladmin", "ping", "-h", "localhost"]
      interval: 10s
      timeout: 5s
      retries: 5

volumes:
  mysql_data:
`
	composePath := filepath.Join(projectPath, "docker-compose.yml")
	return os.WriteFile(composePath, []byte(content), 0644)
}

func (m *Model) StopDockerPostgres(projectPath string) error {
	containerName := fmt.Sprintf("django-stack-%s-*", m.ProjectName)

	cmd := exec.Command("docker", "ps", "-q", "--filter", fmt.Sprintf("name=%s", containerName))
	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	containers := strings.TrimSpace(string(output))
	if containers == "" {
		return nil
	}

	cmd = exec.Command("docker", "stop", containers)
	cmd.Run()

	cmd = exec.Command("docker", "rm", containers)
	cmd.Run()

	return nil
}
