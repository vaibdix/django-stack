package scaffold

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
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
		return "", fmt.Errorf("read randomness for SECRET_KEY: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(key), nil
}

func (e *Env) randomness() io.Reader {
	return e.dependencies().entropy
}

func (e *Env) envFiles() (fileSet, error) {
	content, err := e.renderEnvFileContent()
	if err != nil {
		return nil, err
	}
	files := fileSet{}
	files.add(envFile, content, secretMode)
	return files, nil
}

func (e *Env) envExampleFiles() fileSet {
	files := fileSet{}
	files.add(envExampleFile, e.renderEnvExampleContent(), fileMode)
	return files
}

func (e *Env) writeEnvFile(ctx context.Context) error {
	if e.resolved.SecretKey == "" {
		key, err := generateSecretKey(e.randomness())
		if err != nil {
			return err
		}
		e.resolved.SecretKey = key
	}
	files, err := e.envFiles()
	if err != nil {
		return err
	}
	if err := e.emit(files); err != nil {
		return err
	}

	e.Step("Created: .env (with SECRET_KEY and database config)")
	return nil
}

func (e *Env) writeEnvExampleFile(ctx context.Context) error {
	if err := e.emit(e.envExampleFiles()); err != nil {
		return err
	}

	e.Step("Created: .env.example")
	return nil
}

func (e *Env) envDatabase() (dotenvDatabase, error) {
	name := dbdocker.DatabaseName(e.cfg.Name)

	if !e.cfg.Database.UsesDocker() {
		return dotenvDatabase{}, nil
	}
	if err := requireCredentials(e.resolved.Database.Password, e.resolved.Database.RootPassword, e.resolved.Database.Port); err != nil {
		return dotenvDatabase{}, err
	}
	return dockerDatabase(e.cfg.Database, name, e.resolved.Database.Password, e.resolved.Database.RootPassword, e.resolved.Database.Port), nil
}

func requireCredentials(password, rootPassword, port string) error {
	if password == "" || rootPassword == "" || port == "" {
		return errors.New("the database credentials were not chosen before the .env file was written")
	}
	return nil
}

func (e *Env) envExampleDatabase() dotenvDatabase {
	name := dbdocker.DatabaseName(e.cfg.Name)

	if !e.cfg.Database.UsesDocker() {
		return dotenvDatabase{}
	}
	return dockerDatabase(e.cfg.Database, name, "<set-password>", "<set-root-password>", e.resolved.Database.Port)
}

func (e *Env) renderEnvFileContent() (string, error) {
	database, err := e.envDatabase()
	if err != nil {
		return "", err
	}
	if e.resolved.SecretKey == "" {
		return "", errors.New("no SECRET_KEY was generated for .env")
	}
	return dotenvSection(dotenvDjangoSection(e.resolved.SecretKey), database.render()), nil
}

func (e *Env) renderEnvExampleContent() string {
	return dotenvSection(dotenvDjangoSection(""), e.envExampleDatabase().render())
}
