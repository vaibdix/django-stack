package scaffold

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vaibdix/django-cli/internal/config"
	"github.com/vaibdix/django-cli/internal/scaffold/command/commandtest"
	"github.com/vaibdix/django-cli/internal/scaffold/dbdocker"
)

func TestTheDotEnvAndTheComposeFileShareOneSetOfMySQLCredentials(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	e := &Env{
		Cfg:      config.Project{ProjectName: "shop", DatabaseType: config.MySQLDocker},
		Root:     root,
		Reporter: NopReporter{},
	}
	for _, step := range []func(string) error{e.prepareDatabaseCredentials, e.writeEnvFile, e.writeComposeFile} {
		if err := step(root); err != nil {
			t.Fatalf("preparing the database: %v", err)
		}
	}

	dotenv, err := os.ReadFile(filepath.Join(root, EnvFile))
	if err != nil {
		t.Fatalf("reading .env: %v", err)
	}
	for key, want := range map[string]string{
		"DB_PASSWORD":      e.Cfg.MySQLDockerPassword,
		"DB_ROOT_PASSWORD": e.Cfg.MySQLDockerRootPassword,
		"DB_PORT":          e.Cfg.MySQLDockerPort,
	} {
		if want == "" || !strings.Contains(string(dotenv), key+"="+want+"\n") {
			t.Errorf(".env does not carry the generated %s (%q):\n%s", key, want, dotenv)
		}
	}

	compose, err := os.ReadFile(filepath.Join(root, dbdocker.ComposeFile))
	if err != nil {
		t.Fatalf("reading the compose file: %v", err)
	}
	if !strings.Contains(string(compose), "${DB_ROOT_PASSWORD:?") {
		t.Errorf("the container does not take its root password from .env:\n%s", compose)
	}
}

type collectingReporter struct{ messages []string }

func (r *collectingReporter) Step(message string)     { r.messages = append(r.messages, message) }
func (r *collectingReporter) Progress(message string) { r.messages = append(r.messages, message) }

func TestNoProgressMessageCarriesAGeneratedPassword(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	reporter := &collectingReporter{}
	fake := commandtest.NewFake()
	fake.Responses["docker version --format {{.Server.APIVersion}}"] = []byte("1.47\n")
	e := &Env{
		Cfg:      config.Project{ProjectName: "shop", DatabaseType: config.MySQLDocker},
		Root:     root,
		Runner:   fake,
		Reporter: reporter,
	}
	for _, step := range []func(string) error{e.prepareDatabaseCredentials, e.writeEnvFile, e.writeComposeFile, e.startDatabaseContainer} {
		if err := step(root); err != nil {
			t.Fatalf("starting the database: %v", err)
		}
	}

	for _, message := range reporter.messages {
		for _, secret := range []string{e.Cfg.MySQLDockerPassword, e.Cfg.MySQLDockerRootPassword} {
			if strings.Contains(message, secret) {
				t.Errorf("a progress message shows a generated password: %q", message)
			}
		}
	}
}
