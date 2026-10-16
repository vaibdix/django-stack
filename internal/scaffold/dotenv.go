package scaffold

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vaibdix/django-cli/internal/config"
	"github.com/vaibdix/django-cli/internal/scaffold/dbdocker"
)

const (
	dotenvEmailSection = `# === Email (read by settings.production) ===
# EMAIL_HOST=smtp.gmail.com
# EMAIL_PORT=587
# EMAIL_USE_TLS=True
# EMAIL_HOST_USER=
# EMAIL_HOST_PASSWORD=
# DEFAULT_FROM_EMAIL=webmaster@localhost`

	dotenvSecuritySection = `# === Production security (read by settings.production) ===
# ADMIN_URL=admin/
# CSRF_TRUSTED_ORIGINS=https://example.com
# SECURE_SSL_REDIRECT=True
# SECURE_HSTS_SECONDS=31536000
# SECURE_HSTS_INCLUDE_SUBDOMAINS=False
# SECURE_HSTS_PRELOAD=False
# TRUST_PROXY_SSL_HEADER=False`
)

const (
	defaultPostgresPort = "5432"
	defaultMySQLPort    = "3306"
	mysqlTCPHost        = "127.0.0.1"
)

func dotenvSection(sections ...string) string {
	present := make([]string, 0, len(sections))
	for _, s := range sections {
		if s != "" {
			present = append(present, s)
		}
	}
	return strings.Join(present, "\n\n") + "\n"
}

func dotenvHeader(example bool) string {
	const title = "# Django Stack - Environment Variables"
	if example {
		return title + `
# Copy this file to .env and fill in your values
# NEVER commit .env to version control`
	}
	return title + `
# NEVER commit this file to version control`
}

func secretKeyGuidance() string {
	return "# Required, at least 50 characters. Replace the placeholder below with:\n" +
		"#   python -c 'from django.core.management.utils import get_random_secret_key as k; print(k())'\n"
}

const placeholderSecretKey = "replace-this-placeholder-with-a-generated-50-char-key"

func dotenvDjangoSection(secretKey string) string {
	guidance := ""
	if secretKey == "" {
		guidance = secretKeyGuidance()
		secretKey = placeholderSecretKey
	}

	return `# === Django Settings ===
# local | test | staging | production. This is the only switch for DEBUG.
ENVIRONMENT=local
` + guidance + `SECRET_KEY=` + secretKey + `
ALLOWED_HOSTS=localhost,127.0.0.1`
}

type dotenvVar struct {
	Key   string
	Value string
}

type dotenvDatabase struct {
	Heading string

	Note string

	Vars []dotenvVar
}

func (d dotenvDatabase) render() string {
	if d.Heading == "" {
		return ""
	}

	var b strings.Builder
	b.WriteString(d.Heading)
	if d.Note != "" {
		b.WriteString("\n" + d.Note)
	}
	for _, v := range d.Vars {
		b.WriteString("\n" + v.Key + "=" + v.Value)
	}
	return b.String()
}

func postgresDockerDatabase(name, password, port string) dotenvDatabase {
	return dotenvDatabase{
		Heading: "# === Database (PostgreSQL Docker) ===",
		Vars: []dotenvVar{
			{"DB_NAME", name},
			{"DB_USER", "postgres"},
			{"DB_PASSWORD", password},
			{"DB_HOST", "localhost"},
			{"DB_PORT", port},
		},
	}
}

func mysqlDockerDatabase(name, password, rootPassword, port string) dotenvDatabase {
	return dotenvDatabase{
		Heading: "# === Database (MySQL Docker) ===",
		Vars: []dotenvVar{
			{"DB_NAME", name},
			{"DB_USER", "django_user"},
			{"DB_PASSWORD", password},
			{"DB_ROOT_PASSWORD", rootPassword},
			{"DB_HOST", mysqlTCPHost},
			{"DB_PORT", port},
		},
	}
}

var randRead = func(b []byte) { _, _ = rand.Read(b) }

const secretKeyAlphabet = "abcdefghijklmnopqrstuvwxyz0123456789!@#$%^&*(-_=+)"

func generateSecretKey() string {
	key := make([]byte, 50)
	for i := range key {
		key[i] = secretKeyAlphabet[uniformIndex(len(secretKeyAlphabet))]
	}
	return string(key)
}

func uniformIndex(n int) int {
	limit := 256 - (256 % n)
	buf := make([]byte, 1)
	for {
		randRead(buf)
		if int(buf[0]) < limit {
			return int(buf[0]) % n
		}
	}
}

func generateSecurePassword(length int) string {
	bytes := make([]byte, length)
	randRead(bytes)
	return hex.EncodeToString(bytes)[:length]
}

func portOrDefault(port, fallback string) string {
	if port == "" {
		return fallback
	}
	return port
}

func (e *Env) writeEnvFile(projectPath string) error {
	content := e.renderEnvFileContent()

	envPath := filepath.Join(projectPath, EnvFile)
	if err := os.WriteFile(envPath, []byte(content), 0600); err != nil {
		return fmt.Errorf("failed to create %s file: %w", EnvFile, err)
	}

	e.Step("Created: .env (with SECRET_KEY and database config)")
	return nil
}

func (e *Env) writeEnvExampleFile(projectPath string) error {
	content := e.renderEnvExampleContent()

	envPath := filepath.Join(projectPath, EnvExampleFile)
	if err := os.WriteFile(envPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to create %s file: %w", EnvExampleFile, err)
	}

	e.Step("Created: .env.example")
	return nil
}

func (e *Env) envDatabase() dotenvDatabase {
	name := dbdocker.DatabaseName(e.Cfg.ProjectName)

	switch e.Cfg.DatabaseType {
	case config.PostgresDocker:
		password := e.Cfg.PostgresDockerPassword
		if password == "" {
			password = generateSecurePassword(16)
		}
		return postgresDockerDatabase(name, password, portOrDefault(e.Cfg.PostgresDockerPort, defaultPostgresPort))

	case config.MySQLDocker:
		password := e.Cfg.MySQLDockerPassword
		if password == "" {
			password = generateSecurePassword(16)
		}
		rootPassword := e.Cfg.MySQLDockerRootPassword
		if rootPassword == "" {
			rootPassword = generateSecurePassword(16)
		}
		return mysqlDockerDatabase(name, password, rootPassword,
			portOrDefault(e.Cfg.MySQLDockerPort, defaultMySQLPort))

	default:
		return dotenvDatabase{}
	}
}

func (e *Env) envExampleDatabase() dotenvDatabase {
	name := dbdocker.DatabaseName(e.Cfg.ProjectName)

	switch e.Cfg.DatabaseType {
	case config.PostgresDocker:
		return postgresDockerDatabase(name, "<set-password>",
			portOrDefault(e.Cfg.PostgresDockerPort, defaultPostgresPort))

	case config.MySQLDocker:
		return mysqlDockerDatabase(name, "", "", portOrDefault(e.Cfg.MySQLDockerPort, defaultMySQLPort))

	default:
		return dotenvDatabase{}
	}
}

func (e *Env) renderEnvFileContent() string {

	secretKey := generateSecretKey()
	database := e.envDatabase()

	if database.Heading == "" {
		return dotenvSection(
			dotenvHeader(false),
			dotenvDjangoSection(secretKey),
			dotenvEmailSection,
		)
	}

	return dotenvSection(
		dotenvHeader(false),
		dotenvDjangoSection(secretKey),
		database.render(),
	)
}

func (e *Env) renderEnvExampleContent() string {

	return dotenvSection(
		dotenvHeader(true),
		dotenvDjangoSection(""),
		e.envExampleDatabase().render(),
		dotenvEmailSection,
		e.dotenvSentrySection(),
		dotenvSecuritySection,
	)
}
