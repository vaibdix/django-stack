package dbdocker

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vaibdix/django-stack/internal/config"
	"github.com/vaibdix/django-stack/internal/livetest"
)

func requireLiveDocker(t *testing.T) {
	t.Helper()

	livetest.Require(t, "start real containers")
	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
	defer cancel()
	if err := exec.CommandContext(ctx, "docker", "compose", "version").Run(); err != nil {
		livetest.MissingTool(t, "docker compose is not available: %v", err)
	}
}

func TestTheGeneratedComposeFileStartsAHealthyDatabase(t *testing.T) {
	t.Parallel()

	requireLiveDocker(t)

	cases := map[config.Database]string{

		config.PostgresDocker: "DB_NAME=livecheck\nDB_USER=django_user\nDB_PASSWORD=livecheckpw\nDB_ROOT_PASSWORD=livecheckroot\n",
		config.MySQLDocker:    "DB_NAME=livecheck\nDB_USER=django_user\nDB_PASSWORD=livecheckpw\nDB_ROOT_PASSWORD=livecheckroot\n",
	}

	for db, env := range cases {
		t.Run(string(db), func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			if err := WriteComposeFile(dir, "livecheck", db); err != nil {
				t.Fatalf("WriteComposeFile: %v", err)
			}
			port, err := AvailablePort(55432)
			if err != nil {
				t.Fatalf("AvailablePort: %v", err)
			}
			if err := os.WriteFile(filepath.Join(dir, ".env"), []byte(env+"DB_PORT="+port+"\n"), 0o600); err != nil {
				t.Fatalf("writing .env: %v", err)
			}

			project := fmt.Sprintf("djstacklive%d", time.Now().UnixNano())
			compose := func(ctx context.Context, args ...string) ([]byte, error) {
				cmd := exec.CommandContext(ctx, "docker", append([]string{"compose", "-p", project}, args...)...)
				cmd.Dir = dir
				return cmd.CombinedOutput()
			}
			t.Cleanup(func() {
				ctx, cancel := context.WithTimeout(context.Background(), pullTimeout)
				defer cancel()
				_, _ = compose(ctx, "down", "-v")
			})

			ctx, cancel := context.WithTimeout(context.Background(), pullTimeout)
			defer cancel()
			if out, err := compose(ctx, "up", "-d", "--wait", "--wait-timeout", "180", "db"); err != nil {
				logs, _ := compose(ctx, "logs", "db")
				t.Fatalf("the generated compose file does not start a healthy %s: %v\n%s\n%s",
					db, err, out, tail(string(logs), 40))
			}

			if db == config.PostgresDocker {
				if out, err := compose(ctx, "exec", "-T", "-e", "PGPASSWORD=livecheckpw", Service, "psql", "-h", "127.0.0.1",
					"-U", "django_user", "-d", "livecheck", "-c", "CREATE DATABASE test_livecheck"); err != nil {
					t.Fatalf("the app role cannot create the test database Django's runner needs: %v\n%s", err, out)
				}
			}
			if db == config.MySQLDocker {
				create := "CREATE DATABASE test_livecheck"
				if out, err := compose(ctx, "exec", "-T", Service, "mysql", "-h", "127.0.0.1",
					"-u", "django_user", "-plivecheckpw", "-e", create); err != nil {
					t.Fatalf("the app user cannot create the test database Django's runner needs: %v\n%s", err, out)
				}
			}
		})
	}
}

func tail(text string, lines int) string {
	all := strings.Split(strings.TrimRight(text, "\n"), "\n")
	if len(all) > lines {
		all = all[len(all)-lines:]
	}
	return strings.Join(all, "\n")
}

func TestTheComposeGoldensAreValidComposeFiles(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
	defer cancel()
	if err := exec.CommandContext(ctx, "docker", "compose", "version").Run(); err != nil {
		livetest.MissingTool(t, "docker compose is not installed")
	}

	goldens, err := filepath.Glob(filepath.Join("testdata", "golden", "*.yml"))
	if err != nil || len(goldens) == 0 {
		t.Fatalf("no compose goldens found: %v", err)
	}
	for _, golden := range goldens {
		cmd := exec.CommandContext(ctx, "docker", "compose", "-f", golden, "config", "-q")
		cmd.Env = append(os.Environ(),
			"DB_NAME=check", "DB_USER=check", "DB_PASSWORD=check", "DB_ROOT_PASSWORD=check", "DB_PORT=5432")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Errorf("%s is not a valid compose file, so the golden pins broken output: %v\n%s", golden, err, out)
		}
	}
}
