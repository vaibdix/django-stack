package ui

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func EnvExampleTemplate() string {
	return `# Django Stack - Environment Variables
# Copy this file to .env and fill in your values
# NEVER commit .env to version control

# === Django Settings ===
SECRET_KEY=
DEBUG=True
ALLOWED_HOSTS=localhost,127.0.0.1

# === Email (Optional - uncomment when needed) ===
# EMAIL_BACKEND=django.core.mail.backends.smtp.EmailBackend
# EMAIL_HOST=smtp.gmail.com
# EMAIL_PORT=587
# EMAIL_USE_TLS=True
# EMAIL_HOST_USER=
# EMAIL_HOST_PASSWORD=
# DEFAULT_FROM_EMAIL=webmaster@localhost

# === Cache (Optional - uncomment when needed) ===
# CACHE_URL=redis://localhost:6379/1

# === Production Security (uncomment in production) ===
# SECURE_SSL_REDIRECT=True
# SECURE_HSTS_SECONDS=31536000
`
}

func EnvExampleForMySQL(dbName, dbPort string) string {
	return fmt.Sprintf(`# Django Stack - Environment Variables
# Copy this file to .env and fill in your values
# NEVER commit .env to version control

# === Django Settings ===
SECRET_KEY=
DEBUG=True
ALLOWED_HOSTS=localhost,127.0.0.1

# === Database (MySQL Docker) ===
DB_NAME=%s
DB_USER=django_user
DB_PASSWORD=
DB_ROOT_PASSWORD=
DB_HOST=localhost
DB_PORT=%s

# === Email (Optional - uncomment when needed) ===
# EMAIL_BACKEND=django.core.mail.backends.smtp.EmailBackend
# EMAIL_HOST=smtp.gmail.com
# EMAIL_PORT=587
# EMAIL_USE_TLS=True
# EMAIL_HOST_USER=
# EMAIL_HOST_PASSWORD=
# DEFAULT_FROM_EMAIL=webmaster@localhost

# === Cache (Optional - uncomment when needed) ===
# CACHE_URL=redis://localhost:6379/1

# === Production Security (uncomment in production) ===
# SECURE_SSL_REDIRECT=True
# SECURE_HSTS_SECONDS=31536000
`, dbName, dbPort)
}

func EnvExampleForPostgres(dbName, dbPassword, dbPort string) string {
	return fmt.Sprintf(`# Django Stack - Environment Variables
# Copy this file to .env and fill in your values
# NEVER commit .env to version control

# === Django Settings ===
SECRET_KEY=
DEBUG=True
ALLOWED_HOSTS=localhost,127.0.0.1

# === Database (PostgreSQL Docker) ===
DB_NAME=%s
DB_USER=postgres
DB_PASSWORD=%s
DB_HOST=localhost
DB_PORT=%s

# === Email (Optional - uncomment when needed) ===
# EMAIL_BACKEND=django.core.mail.backends.smtp.EmailBackend
# EMAIL_HOST=smtp.gmail.com
# EMAIL_PORT=587
# EMAIL_USE_TLS=True
# EMAIL_HOST_USER=
# EMAIL_HOST_PASSWORD=
# DEFAULT_FROM_EMAIL=webmaster@localhost

# === Cache (Optional - uncomment when needed) ===
# CACHE_URL=redis://localhost:6379/1

# === Production Security (uncomment in production) ===
# SECURE_SSL_REDIRECT=True
# SECURE_HSTS_SECONDS=31536000
`, dbName, dbPassword, dbPort)
}

func generateSecretKey() string {
	chars := "abcdefghijklmnopqrstuvwxyz0123456789!@#$%^&*(-_=+)"
	key := make([]byte, 50)
	rand.Read(key)
	for i, b := range key {
		key[i] = chars[int(b)%len(chars)]
	}
	return string(key)
}

func generateSecurePassword(length int) string {
	bytes := make([]byte, length)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)[:length]
}

func (m *Model) CreateEnvFile(projectPath string) error {
	content := m.renderEnvFileContent()

	envPath := filepath.Join(projectPath, ".env")
	if err := os.WriteFile(envPath, []byte(content), 0600); err != nil {
		return fmt.Errorf("failed to create .env file: %w", err)
	}

	m.AppendStepMessage("Created: .env (with SECRET_KEY and database config)")
	return nil
}

func (m *Model) CreateEnvExampleFile(projectPath string) error {
	content := m.renderEnvExampleContent()

	envPath := filepath.Join(projectPath, ".env.example")
	if err := os.WriteFile(envPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to create .env.example file: %w", err)
	}

	m.AppendStepMessage("Created: .env.example")
	return nil
}

func (m *Model) renderEnvFileContent() string {
	secretKey := generateSecretKey()

	switch m.DatabaseType {
	case "postgres-docker":
		dbPort := m.PostgresDockerPort
		if dbPort == "" {
			dbPort = "5432"
		}
		dbPassword := m.PostgresDockerPassword
		if dbPassword == "" {
			dbPassword = generateSecurePassword(16)
		}
		dbName := normalizedDatabaseName(m.ProjectName)
		return fmt.Sprintf(`# Django Stack - Environment Variables
# NEVER commit this file to version control

# === Django Settings ===
SECRET_KEY=%s
DEBUG=True
ALLOWED_HOSTS=localhost,127.0.0.1

# === Database (PostgreSQL Docker) ===
DB_NAME=%s
DB_USER=postgres
DB_PASSWORD=%s
DB_HOST=localhost
DB_PORT=%s
`, secretKey, dbName, dbPassword, dbPort)
	case "mysql-docker":
		dbPort := m.MySQLDockerPort
		if dbPort == "" {
			dbPort = "3306"
		}
		dbPassword := m.MySQLDockerPassword
		if dbPassword == "" {
			dbPassword = generateSecurePassword(16)
		}
		rootPassword := generateSecurePassword(16)
		dbName := normalizedDatabaseName(m.ProjectName)
		return fmt.Sprintf(`# Django Stack - Environment Variables
# NEVER commit this file to version control

# === Django Settings ===
SECRET_KEY=%s
DEBUG=True
ALLOWED_HOSTS=localhost,127.0.0.1

# === Database (MySQL Docker) ===
DB_NAME=%s
DB_USER=django_user
DB_PASSWORD=%s
DB_ROOT_PASSWORD=%s
DB_HOST=localhost
DB_PORT=%s
`, secretKey, dbName, dbPassword, rootPassword, dbPort)
	case "postgres-string":
		dbName := normalizedDatabaseName(m.ProjectName)
		return fmt.Sprintf(`# Django Stack - Environment Variables
# NEVER commit this file to version control

# === Django Settings ===
SECRET_KEY=%s
DEBUG=True
ALLOWED_HOSTS=localhost,127.0.0.1

# === Database (PostgreSQL) ===
# Update the password and host below with your PostgreSQL credentials
DB_NAME=%s
DB_USER=postgres
DB_PASSWORD=
DB_HOST=localhost
DB_PORT=5432
`, secretKey, dbName)
	default:
		return fmt.Sprintf(`# Django Stack - Environment Variables
# NEVER commit this file to version control

# === Django Settings ===
SECRET_KEY=%s
DEBUG=True
ALLOWED_HOSTS=localhost,127.0.0.1

# === Email (Optional - uncomment when needed) ===
# EMAIL_BACKEND=django.core.mail.backends.smtp.EmailBackend
# EMAIL_HOST=smtp.gmail.com
# EMAIL_PORT=587
# EMAIL_USE_TLS=True
# EMAIL_HOST_USER=
# EMAIL_HOST_PASSWORD=
# DEFAULT_FROM_EMAIL=webmaster@localhost

# === Cache (Optional - uncomment when needed) ===
# CACHE_URL=redis://localhost:6379/1
`, secretKey)
	}
}

func (m *Model) renderEnvExampleContent() string {
	switch m.DatabaseType {
	case "postgres-docker":
		dbPort := m.PostgresDockerPort
		if dbPort == "" {
			dbPort = "5432"
		}
		return EnvExampleForPostgres(normalizedDatabaseName(m.ProjectName), "<set-password>", dbPort)
	case "mysql-docker":
		dbPort := m.MySQLDockerPort
		if dbPort == "" {
			dbPort = "3306"
		}
		return EnvExampleForMySQL(normalizedDatabaseName(m.ProjectName), dbPort)
	case "postgres-string":
		return fmt.Sprintf(`# Django Stack - Environment Variables
# Copy this file to .env and fill in your values
# NEVER commit .env to version control

# === Django Settings ===
SECRET_KEY=
DEBUG=True
ALLOWED_HOSTS=localhost,127.0.0.1

# === Database (PostgreSQL) ===
# Update the password and host below with your PostgreSQL credentials
DB_NAME=%s
DB_USER=postgres
DB_PASSWORD=
DB_HOST=localhost
DB_PORT=5432

# === Email (Optional - uncomment when needed) ===
# EMAIL_BACKEND=django.core.mail.backends.smtp.EmailBackend
# EMAIL_HOST=smtp.gmail.com
# EMAIL_PORT=587
# EMAIL_USE_TLS=True
# EMAIL_HOST_USER=
# EMAIL_HOST_PASSWORD=
# DEFAULT_FROM_EMAIL=webmaster@localhost

# === Cache (Optional - uncomment when needed) ===
# CACHE_URL=redis://localhost:6379/1

# === Production Security (uncomment in production) ===
# SECURE_SSL_REDIRECT=True
# SECURE_HSTS_SECONDS=31536000
`, normalizedDatabaseName(m.ProjectName))
	default:
		return EnvExampleTemplate()
	}
}

func normalizedDatabaseName(projectName string) string {
	name := filepath.Base(filepath.Clean(projectName))
	return strings.ToLower(strings.ReplaceAll(name, "-", "_"))
}
