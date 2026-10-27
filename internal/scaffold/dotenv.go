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

func (e *Env) randomness() io.Reader { return e.machine().Rand }

func (p project) envFiles() (fileSet, error) {
	content, err := p.renderEnvFileContent()
	if err != nil {
		return nil, err
	}
	files := fileSet{}
	files.add(envFile, content, secretMode)
	return files, nil
}

func (p project) envExampleFiles() fileSet {
	files := fileSet{}
	files.add(envExampleFile, p.renderEnvExampleContent(), fileMode)
	return files
}

func (e *Env) resolveSecrets() error {
	if e.resolved.SecretKey == "" {
		key, err := generateSecretKey(e.randomness())
		if err != nil {
			return err
		}
		e.resolved.SecretKey = key
	}
	if e.cfg.Database().UsesDocker() && e.resolved.Database == (config.DatabaseCredentials{}) {
		return e.prepareDatabaseCredentials()
	}
	return nil
}

func (e *Env) writeEnvFile(ctx context.Context) error {
	if err := e.emitGroup(groupEnv); err != nil {
		return err
	}

	e.Step("Created: .env (with SECRET_KEY and database config)")
	return nil
}

func (e *Env) writeEnvExampleFile(ctx context.Context) error {
	if err := e.emitGroup(groupEnvExample); err != nil {
		return err
	}

	e.Step("Created: .env.example")
	return nil
}

func (p project) envDatabase() (dotenvDatabase, error) {
	name := dbdocker.DatabaseName(p.cfg.Name().String())

	if !p.cfg.Database().UsesDocker() {
		return dotenvDatabase{}, nil
	}
	if err := requireCredentials(p.resolved.Database.Password, p.resolved.Database.RootPassword, p.resolved.Database.Port); err != nil {
		return dotenvDatabase{}, err
	}
	return dockerDatabase(p.cfg.Database(), name, p.resolved.Database.Password, p.resolved.Database.RootPassword, p.resolved.Database.Port), nil
}

func requireCredentials(password, rootPassword, port string) error {
	if password == "" || rootPassword == "" || port == "" {
		return errors.New("the database credentials were not chosen before the .env file was written")
	}
	return nil
}

func (p project) envExampleDatabase() dotenvDatabase {
	name := dbdocker.DatabaseName(p.cfg.Name().String())

	if !p.cfg.Database().UsesDocker() {
		return dotenvDatabase{}
	}
	return dockerDatabase(p.cfg.Database(), name, "<set-password>", "<set-root-password>", p.resolved.Database.Port)
}

func (p project) renderEnvFileContent() (string, error) {
	database, err := p.envDatabase()
	if err != nil {
		return "", err
	}
	if p.resolved.SecretKey == "" {
		return "", errors.New("no SECRET_KEY was generated for .env")
	}
	return dotenvSection(dotenvDjangoSection(p.resolved.SecretKey), database.render()), nil
}

func (p project) renderEnvExampleContent() string {
	return dotenvSection(dotenvDjangoSection(""), p.envExampleDatabase().render())
}
