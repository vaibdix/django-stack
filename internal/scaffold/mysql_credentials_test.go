package scaffold

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vaibdix/django-cli/internal/config"
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
