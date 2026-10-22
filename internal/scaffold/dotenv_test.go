package scaffold

import (
	"strings"
	"testing"

	"github.com/vaibdix/django-stack/internal/config"
)

func TestADockerEnvFileWithoutChosenCredentialsIsRefused(t *testing.T) {
	t.Parallel()

	for _, db := range []config.Database{config.PostgresDocker, config.MySQLDocker} {
		e := &Env{cfg: config.Project{ProjectName: "demo", DatabaseType: db}}
		if content, err := e.renderEnvFileContent(); err == nil {
			t.Errorf("%s: .env was rendered with credentials nobody chose:\n%s", db, content)
		}
	}
}

func TestRenderEnvFileContentSQLiteKeepsDefaultDBName(t *testing.T) {
	t.Parallel()

	env := &Env{
		cfg: config.Project{
			ProjectName:  "demo-project",
			DatabaseType: config.SQLite,
		},
	}

	content := envFileFor(t, env)

	if strings.Contains(content, "DB_ENGINE") || strings.Contains(content, "DB_NAME") {
		t.Fatalf("sqlite env content should not have DB_ENGINE or DB_NAME (uses hardcoded settings), got:\n%s", content)
	}
	if strings.Contains(content, "DATABASE_URL=postgres") {
		t.Fatalf("sqlite env content should not have postgres DATABASE_URL, got:\n%s", content)
	}
}

func TestRenderEnvFileContentPostgresIsExplicit(t *testing.T) {
	t.Parallel()

	env := &Env{
		cfg: withCredentials(config.Project{
			ProjectName:            "demo-project",
			DatabaseType:           config.PostgresDocker,
			PostgresDockerPort:     "5544",
			PostgresDockerPassword: "secretpass",
		}),
	}

	content := envFileFor(t, env)

	checks := map[string]bool{
		"DB_NAME=demo_project":   strings.Contains(content, "DB_NAME=demo_project"),
		"DB_USER=django_user":    strings.Contains(content, "DB_USER=django_user"),
		"DB_PASSWORD=secretpass": strings.Contains(content, "DB_PASSWORD=secretpass"),
		"DB_PORT=5544":           strings.Contains(content, "DB_PORT=5544"),
	}
	for key, found := range checks {
		if !found {
			t.Fatalf("expected postgres env content to contain %s, got:\n%s", key, content)
		}
	}
}
