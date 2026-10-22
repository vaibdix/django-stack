package scaffold

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/vaibdix/django-stack/internal/config"
	"github.com/vaibdix/django-stack/internal/dbdocker"
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

func dockerDatabase(db config.Database, name, password, rootPassword, port string) dotenvDatabase {
	return dotenvDatabase{
		Vars: []dotenvVar{
			{"DB_NAME", name},
			{"DB_USER", "django_user"},
			{"DB_PASSWORD", password},
			{"DB_ROOT_PASSWORD", rootPassword},
			{"DB_HOST", db.Spec().Host},
			{"DB_PORT", port},
		},
	}
}

const secretKeyBytes = 38

func generateSecretKey(entropy io.Reader) (string, error) {
	key := make([]byte, secretKeyBytes)
	if _, err := io.ReadFull(entropy, key); err != nil {
		return "", fmt.Errorf("failed to read randomness for SECRET_KEY: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(key), nil
}

func (e *Env) randomness() io.Reader {
	if e.entropy == nil {
		return rand.Reader
	}
	return e.entropy
}

func (e *Env) writeEnvFile(projectPath string) error {
	content, err := e.renderEnvFileContent()
	if err != nil {
		return err
	}

	envPath := filepath.Join(projectPath, envFile)
	if err := os.WriteFile(envPath, []byte(content), 0600); err != nil {
		return fmt.Errorf("failed to create %s file: %w", envFile, err)
	}

	e.Step("Created: .env (with SECRET_KEY and database config)")
	return nil
}

func (e *Env) writeEnvExampleFile(projectPath string) error {
	content := e.renderEnvExampleContent()

	envPath := filepath.Join(projectPath, envExampleFile)
	if err := os.WriteFile(envPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to create %s file: %w", envExampleFile, err)
	}

	e.Step("Created: .env.example")
	return nil
}

func (e *Env) envDatabase() (dotenvDatabase, error) {
	name := dbdocker.DatabaseName(e.cfg.ProjectName)

	if !e.cfg.DatabaseType.UsesDocker() {
		return dotenvDatabase{}, nil
	}
	if err := requireCredentials(e.cfg.DatabasePassword, e.cfg.DatabaseRootPassword, e.cfg.DatabasePort); err != nil {
		return dotenvDatabase{}, err
	}
	return dockerDatabase(e.cfg.DatabaseType, name, e.cfg.DatabasePassword, e.cfg.DatabaseRootPassword, e.cfg.DatabasePort), nil
}

func requireCredentials(password, rootPassword, port string) error {
	if password == "" || rootPassword == "" || port == "" {
		return errors.New("the database credentials were not chosen before the .env file was written")
	}
	return nil
}

func (e *Env) envExampleDatabase() dotenvDatabase {
	name := dbdocker.DatabaseName(e.cfg.ProjectName)

	if !e.cfg.DatabaseType.UsesDocker() {
		return dotenvDatabase{}
	}
	return dockerDatabase(e.cfg.DatabaseType, name, "<set-password>", "<set-root-password>", e.cfg.DatabasePort)
}

func (e *Env) renderEnvFileContent() (string, error) {
	database, err := e.envDatabase()
	if err != nil {
		return "", err
	}
	secretKey, err := generateSecretKey(e.randomness())
	if err != nil {
		return "", err
	}
	return dotenvSection(dotenvDjangoSection(secretKey), database.render()), nil
}

func (e *Env) renderEnvExampleContent() string {
	return dotenvSection(dotenvDjangoSection(""), e.envExampleDatabase().render())
}
