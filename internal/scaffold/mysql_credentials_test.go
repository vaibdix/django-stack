package scaffold

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vaibdix/django-stack/internal/command/commandtest"
	"github.com/vaibdix/django-stack/internal/config"
	"github.com/vaibdix/django-stack/internal/dbdocker"
)

func TestTheDotEnvAndTheComposeFileShareOneSetOfMySQLCredentials(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	e := &Env{
		cfg:      config.Project{ProjectName: "shop", DatabaseType: config.MySQLDocker},
		root:     root,
		reporter: NopReporter{},
	}
	for _, step := range []func(string) error{e.prepareDatabaseCredentials, e.writeEnvFile, e.writeComposeFile} {
		if err := step(root); err != nil {
			t.Fatalf("preparing the database: %v", err)
		}
	}

	dotenv, err := os.ReadFile(filepath.Join(root, envFile))
	if err != nil {
		t.Fatalf("reading .env: %v", err)
	}
	for key, want := range map[string]string{
		"DB_PASSWORD":      e.cfg.DatabasePassword,
		"DB_ROOT_PASSWORD": e.cfg.DatabaseRootPassword,
		"DB_PORT":          e.cfg.DatabasePort,
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
		cfg:      config.Project{ProjectName: "shop", DatabaseType: config.MySQLDocker},
		root:     root,
		runner:   fake,
		reporter: reporter,
	}
	for _, step := range []func(string) error{e.prepareDatabaseCredentials, e.writeEnvFile, e.writeComposeFile, e.startDatabaseContainer} {
		if err := step(root); err != nil {
			t.Fatalf("starting the database: %v", err)
		}
	}

	for _, message := range reporter.messages {
		for _, secret := range []string{e.cfg.DatabasePassword, e.cfg.DatabaseRootPassword} {
			if strings.Contains(message, secret) {
				t.Errorf("a progress message shows a generated password: %q", message)
			}
		}
	}
}
