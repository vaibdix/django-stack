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

// Environment-file rendering.
//
// A .env and a .env.example are the same document with different values and a
// different set of optional sections. Previously they were eight hand-written
// literals — four database variants times two file kinds — which is why the
// Django-settings header appeared eight times and the email/cache boilerplate
// five. Editing one meant remembering to edit the others.
//
// Here a file is a list of sections joined by a blank line, and each database
// backend is one descriptor. Adding a backend is a descriptor, not a copy.

// Fixed sections. These are byte-for-byte what the old literals emitted.
const (
	dotenvEmailSection = `# === Email (Optional - uncomment when needed) ===
# EMAIL_BACKEND=django.core.mail.backends.smtp.EmailBackend
# EMAIL_HOST=smtp.gmail.com
# EMAIL_PORT=587
# EMAIL_USE_TLS=True
# EMAIL_HOST_USER=
# EMAIL_HOST_PASSWORD=
# DEFAULT_FROM_EMAIL=webmaster@localhost`

	dotenvCacheSection = `# === Cache (Optional - uncomment when needed) ===
# CACHE_URL=redis://localhost:6379/1`

	dotenvSecuritySection = `# === Production Security (uncomment in production) ===
# SECURE_SSL_REDIRECT=True
# SECURE_HSTS_SECONDS=31536000`
)

// Default ports, used when no container has been started yet.
const (
	defaultPostgresPort = "5432"
	defaultMySQLPort    = "3306"
)

// dotenvSection joins the sections of an environment file.
//
// Sections are separated by one blank line and the file ends with a newline,
// which is the layout every previous literal used.
func dotenvSection(sections ...string) string {
	present := make([]string, 0, len(sections))
	for _, s := range sections {
		if s != "" {
			present = append(present, s)
		}
	}
	return strings.Join(present, "\n\n") + "\n"
}

// dotenvHeader is the comment block at the top of the file. The .env and
// .env.example variants differ only in whether they tell you to copy the file.
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

// dotenvDjangoSection renders the core Django variables. secretKey is empty in
// a .env.example, which is the only difference between the two kinds.
func dotenvDjangoSection(secretKey string) string {
	return `# === Django Settings ===
SECRET_KEY=` + secretKey + `
DEBUG=True
ALLOWED_HOSTS=localhost,127.0.0.1`
}

// dotenvVar is one KEY=value line. A slice preserves declaration order, which
// a map would not.
type dotenvVar struct {
	Key   string
	Value string
}

// dotenvDatabase describes the database block for one backend.
type dotenvDatabase struct {
	// Heading labels the section, e.g. "# === Database (PostgreSQL Docker) ===".
	Heading string
	// Note is an optional comment line below the heading.
	Note string
	// Vars are the variables, in the order they are written.
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
			{"DB_HOST", "localhost"},
			{"DB_PORT", port},
		},
	}
}

// postgresStringDatabase targets an externally managed server, so the password
// is always left for the user to fill in.
func postgresStringDatabase(name string) dotenvDatabase {
	return dotenvDatabase{
		Heading: "# === Database (PostgreSQL) ===",
		Note:    "# Update the password and host below with your PostgreSQL credentials",
		Vars: []dotenvVar{
			{"DB_NAME", name},
			{"DB_USER", "postgres"},
			{"DB_PASSWORD", ""},
			{"DB_HOST", "localhost"},
			{"DB_PORT", defaultPostgresPort},
		},
	}
}

// randRead is the single source of randomness for generated secrets. It is a
// package-level var so golden tests can substitute a deterministic reader;
// production always uses crypto/rand.
//
// crypto/rand.Read is documented never to return an error, so the result is
// discarded explicitly rather than plumbed through every generator signature.
var randRead = func(b []byte) { _, _ = rand.Read(b) }

func generateSecretKey() string {
	chars := "abcdefghijklmnopqrstuvwxyz0123456789!@#$%^&*(-_=+)"
	key := make([]byte, 50)
	randRead(key)
	for i, b := range key {
		key[i] = chars[int(b)%len(chars)]
	}
	return string(key)
}

func generateSecurePassword(length int) string {
	bytes := make([]byte, length)
	randRead(bytes)
	return hex.EncodeToString(bytes)[:length]
}

// portOrDefault falls back to the backend's standard port when no container
// has recorded one.
func portOrDefault(port, fallback string) string {
	if port == "" {
		return fallback
	}
	return port
}

// writeEnvFile writes the project's .env, including a generated SECRET_KEY and
// the credentials of any database container that was started.
func (e *Env) writeEnvFile(projectPath string) error {
	content := e.renderEnvFileContent()

	envPath := filepath.Join(projectPath, EnvFile)
	if err := os.WriteFile(envPath, []byte(content), 0600); err != nil {
		return fmt.Errorf("failed to create %s file: %w", EnvFile, err)
	}

	e.Step("Created: .env (with SECRET_KEY and database config)")
	return nil
}

// writeEnvExampleFile writes the committable .env.example: the same shape as
// .env with every secret blanked out.
func (e *Env) writeEnvExampleFile(projectPath string) error {
	content := e.renderEnvExampleContent()

	envPath := filepath.Join(projectPath, EnvExampleFile)
	if err := os.WriteFile(envPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to create %s file: %w", EnvExampleFile, err)
	}

	e.Step("Created: .env.example")
	return nil
}

// envDatabase builds the database section for the configured backend.
//
// The password is reused from the container this run started, if there was one.
// That coupling is deliberate and load-bearing: writing .env before the
// container exists would put a password in the file that the server does not
// have. Generate documents the ordering; this is the read side of it.
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
		return mysqlDockerDatabase(name, password, generateSecurePassword(16),
			portOrDefault(e.Cfg.MySQLDockerPort, defaultMySQLPort))

	case config.PostgresString:
		return postgresStringDatabase(name)

	default:
		return dotenvDatabase{}
	}
}

// envExampleDatabase is envDatabase with the secrets stripped.
func (e *Env) envExampleDatabase() dotenvDatabase {
	name := dbdocker.DatabaseName(e.Cfg.ProjectName)

	switch e.Cfg.DatabaseType {
	case config.PostgresDocker:
		return postgresDockerDatabase(name, "<set-password>",
			portOrDefault(e.Cfg.PostgresDockerPort, defaultPostgresPort))

	case config.MySQLDocker:
		return mysqlDockerDatabase(name, "", "", portOrDefault(e.Cfg.MySQLDockerPort, defaultMySQLPort))

	case config.PostgresString:
		return postgresStringDatabase(name)

	default:
		return dotenvDatabase{}
	}
}

func (e *Env) renderEnvFileContent() string {
	// The SECRET_KEY is drawn before the database credentials. Both come from
	// the same randomness source, so the order is observable in the golden
	// files; keeping it fixed means a change to the generated secrets shows up
	// as a real diff rather than as noise from a reordered call.
	secretKey := generateSecretKey()
	database := e.envDatabase()

	// SQLite needs no database variables, so its .env carries the optional
	// email and cache hints instead — those would only be noise alongside a
	// real database block the user still has to configure.
	if database.Heading == "" {
		return dotenvSection(
			dotenvHeader(false),
			dotenvDjangoSection(secretKey),
			dotenvEmailSection,
			dotenvCacheSection,
		)
	}

	return dotenvSection(
		dotenvHeader(false),
		dotenvDjangoSection(secretKey),
		database.render(),
	)
}

func (e *Env) renderEnvExampleContent() string {
	// The example file always carries every optional section: it is the
	// reference a user copies from.
	return dotenvSection(
		dotenvHeader(true),
		dotenvDjangoSection(""),
		e.envExampleDatabase().render(),
		dotenvEmailSection,
		dotenvCacheSection,
		dotenvSecuritySection,
	)
}
