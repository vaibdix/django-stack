package scaffold

import (
	"crypto/rand"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vaibdix/django-cli/internal/config"
	"github.com/vaibdix/django-cli/internal/scaffold/dbdocker"
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
	return strings.Join(present, "\n") + "\n"
}

const placeholderSecretKey = "replace-this-placeholder-with-a-generated-50-char-key"

func dotenvDjangoSection(secretKey string) string {
	if secretKey == "" {
		secretKey = placeholderSecretKey
	}
	return "ENVIRONMENT=local\nSECRET_KEY=" + secretKey + "\nALLOWED_HOSTS=localhost,127.0.0.1"
}

type dotenvVar struct {
	Key   string
	Value string
}

type dotenvDatabase struct {
	Vars []dotenvVar
}

func (d dotenvDatabase) render() string {
	lines := make([]string, 0, len(d.Vars))
	for _, v := range d.Vars {
		lines = append(lines, v.Key+"="+v.Value)
	}
	return strings.Join(lines, "\n")
}

func postgresDockerDatabase(name, password, rootPassword, port string) dotenvDatabase {
	return dotenvDatabase{
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

func mysqlDockerDatabase(name, password, rootPassword, port string) dotenvDatabase {
	return dotenvDatabase{
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

const secretKeyAlphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"

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

func (e *Env) writeEnvFile(projectPath string) error {
	content, err := e.renderEnvFileContent()
	if err != nil {
		return err
	}

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

func (e *Env) envDatabase() (dotenvDatabase, error) {
	name := dbdocker.DatabaseName(e.Cfg.ProjectName)

	switch e.Cfg.DatabaseType {
	case config.PostgresDocker:
		if err := requireCredentials(e.Cfg.PostgresDockerPassword, e.Cfg.PostgresDockerRootPassword, e.Cfg.PostgresDockerPort); err != nil {
			return dotenvDatabase{}, err
		}
		return postgresDockerDatabase(name, e.Cfg.PostgresDockerPassword, e.Cfg.PostgresDockerRootPassword, e.Cfg.PostgresDockerPort), nil

	case config.MySQLDocker:
		if err := requireCredentials(e.Cfg.MySQLDockerPassword, e.Cfg.MySQLDockerRootPassword, e.Cfg.MySQLDockerPort); err != nil {
			return dotenvDatabase{}, err
		}
		return mysqlDockerDatabase(name, e.Cfg.MySQLDockerPassword, e.Cfg.MySQLDockerRootPassword, e.Cfg.MySQLDockerPort), nil

	default:
		return dotenvDatabase{}, nil
	}
}

func requireCredentials(password, rootPassword, port string) error {
	if password == "" || rootPassword == "" || port == "" {
		return errors.New("the database credentials were not chosen before the .env file was written")
	}
	return nil
}

func (e *Env) envExampleDatabase() dotenvDatabase {
	name := dbdocker.DatabaseName(e.Cfg.ProjectName)

	switch e.Cfg.DatabaseType {
	case config.PostgresDocker:
		return postgresDockerDatabase(name, "<set-password>", "<set-root-password>", e.Cfg.PostgresDockerPort)

	case config.MySQLDocker:
		return mysqlDockerDatabase(name, "<set-password>", "<set-root-password>", e.Cfg.MySQLDockerPort)

	default:
		return dotenvDatabase{}
	}
}

func (e *Env) renderEnvFileContent() (string, error) {
	database, err := e.envDatabase()
	if err != nil {
		return "", err
	}
	return dotenvSection(dotenvDjangoSection(generateSecretKey()), database.render()), nil
}

func (e *Env) renderEnvExampleContent() string {
	return dotenvSection(dotenvDjangoSection(""), e.envExampleDatabase().render())
}
