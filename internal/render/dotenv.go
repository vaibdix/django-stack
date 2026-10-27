package render

import (
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

func GenerateSecretKey(entropy io.Reader) (string, error) {
	key := make([]byte, secretKeyBytes)
	if _, err := io.ReadFull(entropy, key); err != nil {
		return "", fmt.Errorf("read randomness for SECRET_KEY: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(key), nil
}

func (p Project) envFiles() (FileSet, error) {
	content, err := p.renderEnvFileContent()
	if err != nil {
		return nil, err
	}
	files := FileSet{}
	files.add(EnvFile, content, secretMode)
	return files, nil
}

func (p Project) envExampleFiles() FileSet {
	files := FileSet{}
	files.add(EnvExampleFile, p.renderEnvExampleContent(), FileMode)
	return files
}

func (p Project) envDatabase() (dotenvDatabase, error) {
	name := dbdocker.DatabaseName(p.Cfg.Name().String())

	if !p.Cfg.Database().UsesDocker() {
		return dotenvDatabase{}, nil
	}
	if err := requireCredentials(p.Resolved.Database.Password, p.Resolved.Database.RootPassword, p.Resolved.Database.Port); err != nil {
		return dotenvDatabase{}, err
	}
	return dockerDatabase(p.Cfg.Database(), name, p.Resolved.Database.Password, p.Resolved.Database.RootPassword, p.Resolved.Database.Port), nil
}

func requireCredentials(password, rootPassword, port string) error {
	if password == "" || rootPassword == "" || port == "" {
		return errors.New("the database credentials were not chosen before the .env file was written")
	}
	return nil
}

func (p Project) envExampleDatabase() dotenvDatabase {
	name := dbdocker.DatabaseName(p.Cfg.Name().String())

	if !p.Cfg.Database().UsesDocker() {
		return dotenvDatabase{}
	}
	return dockerDatabase(p.Cfg.Database(), name, "<set-password>", "<set-root-password>", p.Resolved.Database.Port)
}

func (p Project) renderEnvFileContent() (string, error) {
	database, err := p.envDatabase()
	if err != nil {
		return "", err
	}
	if p.Resolved.SecretKey == "" {
		return "", errors.New("no SECRET_KEY was generated for .env")
	}
	return dotenvSection(dotenvDjangoSection(p.Resolved.SecretKey), database.render()), nil
}

func (p Project) renderEnvExampleContent() string {
	return dotenvSection(dotenvDjangoSection(""), p.envExampleDatabase().render())
}
