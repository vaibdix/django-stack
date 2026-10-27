package scaffold

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vaibdix/django-stack/internal/command/commandtest"
	"github.com/vaibdix/django-stack/internal/config"
	"github.com/vaibdix/django-stack/internal/dbdocker"
	"github.com/vaibdix/django-stack/internal/port"
	"github.com/vaibdix/django-stack/internal/render"
)

func TestTheDotEnvAndTheComposeFileShareOneSetOfMySQLCredentials(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	e := &Env{
		Cfg:  mustSpec(config.Choices{Name: "shop", Database: config.MySQLDocker}),
		root: root,
		dir:  root,
	}
	for _, step := range []func(context.Context) error{func(context.Context) error { return e.resolveSecrets() }, e.writeEnvFile, e.writeComposeFile} {
		if err := step(t.Context()); err != nil {
			t.Fatalf("preparing the database: %v", err)
		}
	}

	dotenv, err := os.ReadFile(filepath.Join(root, render.EnvFile))
	if err != nil {
		t.Fatalf("reading .env: %v", err)
	}
	for key, want := range map[string]string{
		"DB_PASSWORD":      e.Resolved.Database.Password,
		"DB_ROOT_PASSWORD": e.Resolved.Database.RootPassword,
		"DB_PORT":          e.Resolved.Database.Port,
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

func TestNoProgressMessageCarriesAGeneratedPassword(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	fake := commandtest.NewFake()
	fake.Responses["docker version --format {{.Server.APIVersion}}"] = []byte("1.47\n")
	e := &Env{
		Cfg:   mustSpec(config.Choices{Name: "shop", Database: config.MySQLDocker}),
		root:  root,
		dir:   root,
		ports: port.Ports{Run: fake},
	}
	messages := recordMessages(e)
	for _, step := range []func(context.Context) error{func(context.Context) error { return e.resolveSecrets() }, e.writeEnvFile, e.writeComposeFile, e.startDatabaseContainer} {
		if err := step(t.Context()); err != nil {
			t.Fatalf("starting the database: %v", err)
		}
	}

	for _, message := range messages() {
		for _, secret := range []string{e.Resolved.Database.Password, e.Resolved.Database.RootPassword} {
			if strings.Contains(message, secret) {
				t.Errorf("a progress message shows a generated password: %q", message)
			}
		}
	}
}
