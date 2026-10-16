package scaffold

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vaibdix/django-cli/internal/config"
	"github.com/vaibdix/django-cli/internal/scaffold/command/commandtest"
)

func dockerRunEnv(t *testing.T, fake *commandtest.Fake, key string) string {
	t.Helper()

	for _, call := range fake.Calls() {
		if call.Name != "docker" || len(call.Args) == 0 || call.Args[0] != "run" {
			continue
		}
		for _, arg := range call.Args {
			if value, ok := strings.CutPrefix(arg, key+"="); ok {
				return value
			}
		}
	}
	t.Fatalf("docker run never received %s; calls: %v", key, fake.CommandLines())
	return ""
}

func TestTheDotEnvCarriesTheRootPasswordTheMySQLContainerStartedWith(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	fake := commandtest.NewFake()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	e := &Env{
		Cfg:      config.Project{ProjectName: "shop", DatabaseType: config.MySQLDocker},
		Root:     root,
		Runner:   fake,
		Reporter: NopReporter{},
		ctx:      ctx,
	}
	if err := e.startDatabaseContainer(root); err != nil {
		t.Fatalf("startDatabaseContainer: %v", err)
	}
	if err := e.writeEnvFile(root); err != nil {
		t.Fatalf("writeEnvFile: %v", err)
	}

	dotenv, err := os.ReadFile(filepath.Join(root, EnvFile))
	if err != nil {
		t.Fatalf("reading .env: %v", err)
	}
	want := dockerRunEnv(t, fake, "MYSQL_ROOT_PASSWORD")
	if !strings.Contains(string(dotenv), "DB_ROOT_PASSWORD="+want+"\n") {
		t.Errorf(".env does not carry the root password the container was started with (%s):\n%s",
			want, dotenv)
	}
}
