package scaffold

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vaibdix/django-stack/internal/config"
	"github.com/vaibdix/django-stack/internal/report"
)

const confRuntimeTimeout = 3 * time.Minute

func uvRunPydantic(t *testing.T, dir, script string) (string, bool) {
	t.Helper()

	uv := requireUV(t)

	ctx, cancel := context.WithTimeout(context.Background(), confRuntimeTimeout)
	t.Cleanup(cancel)

	cmd := exec.CommandContext(ctx, uv, "run", "--no-project",
		"--with", "pydantic-settings", "--with", "django", "python", "-c", script)
	cmd.Dir = dir
	cmd.Env = environmentWithout(os.Environ(),
		"SECRET_KEY", "DEBUG", "ALLOWED_HOSTS", "DB_NAME", "DB_USER", "DB_PASSWORD", "DB_HOST", "DB_PORT")

	out, runErr := cmd.CombinedOutput()
	return string(out), runErr == nil
}

func environmentWithout(environment []string, names ...string) []string {
	drop := make(map[string]bool, len(names))
	for _, name := range names {
		drop[name] = true
	}

	kept := make([]string, 0, len(environment))
	for _, entry := range environment {
		name, _, _ := strings.Cut(entry, "=")
		if !drop[name] {
			kept = append(kept, entry)
		}
	}
	return kept
}

func writeTypedProject(t *testing.T, cfg config.Choices, preset config.DatabaseCredentials) (root string, e *Env) {
	t.Helper()

	root = t.TempDir()
	e = &Env{cfg: cfg, resolved: credentialsFor(cfg.Database, preset), reporter: report.Nop{}}

	if err := os.MkdirAll(filepath.Join(root, cfg.ModuleName()), 0o755); err != nil {
		t.Fatalf("creating the module directory: %v", err)
	}
	if err := e.at(root).writeConfModule(); err != nil {
		t.Fatalf("writing conf.py: %v", err)
	}
	e.dir = root
	if err := e.writeEnvFile(t.Context()); err != nil {
		t.Fatalf("writing .env: %v", err)
	}
	return root, e
}

func typedProjectConfig() config.Choices {
	return config.Choices{
		Name:     "demo",
		Django:   "6.1",
		Database: config.SQLite,
		AddOns:   config.NewSet(config.Pydantic),
	}
}

func TestGeneratedSettingsParseTheDotEnvAsWritten(t *testing.T) {
	t.Parallel()

	root, _ := writeTypedProject(t, typedProjectConfig(), config.DatabaseCredentials{})

	out, ok := uvRunPydantic(t, root, `
import sys
sys.path.insert(0, ".")
from demo.conf import env

assert env.ALLOWED_HOSTS == ["localhost", "127.0.0.1"], env.ALLOWED_HOSTS
assert env.ENVIRONMENT == "local", env.ENVIRONMENT
assert env.DEBUG is True, env.DEBUG
assert len(env.SECRET_KEY.get_secret_value()) >= 50, "SECRET_KEY did not load"
assert env.SECRET_KEY.get_secret_value() not in repr(env), "the secret leaks through repr"
print("OK")
`)

	if !ok {
		t.Fatalf("the generated conf.py cannot read the .env the scaffold writes:\n%s", out)
	}
	if !strings.Contains(out, "OK") {
		t.Errorf("assertions did not run:\n%s", out)
	}
}

func TestTypedSettingsFailLoudlyOnAMissingSecretKey(t *testing.T) {
	t.Parallel()

	root, _ := writeTypedProject(t, typedProjectConfig(), config.DatabaseCredentials{})

	envPath := filepath.Join(root, envFile)
	contents, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatalf("reading the generated .env: %v", err)
	}

	var kept []string
	for _, line := range strings.Split(string(contents), "\n") {
		if !strings.HasPrefix(line, "SECRET_KEY=") {
			kept = append(kept, line)
		}
	}
	if err := os.WriteFile(envPath, []byte(strings.Join(kept, "\n")), 0o600); err != nil {
		t.Fatalf("rewriting .env: %v", err)
	}

	out, ok := uvRunPydantic(t, root, `
import sys
sys.path.insert(0, ".")
import demo.conf
`)

	if ok {
		t.Fatalf("a project with no SECRET_KEY started anyway:\n%s", out)
	}
	if !strings.Contains(out, "SECRET_KEY") {
		t.Errorf("the failure does not name the missing field, so it is not actionable:\n%s", out)
	}
}

func TestTypedSettingsRejectValuesThatAreTheRightTypeButWrong(t *testing.T) {
	t.Parallel()

	cfg := typedProjectConfig()
	cfg.Database = config.PostgresDocker
	preset := config.DatabaseCredentials{Password: "pw"}

	for _, tc := range []struct {
		name  string
		key   string
		value string
	}{
		{name: "empty secret key", key: "SECRET_KEY", value: ""},
		{name: "short secret key", key: "SECRET_KEY", value: "too-short-to-be-a-django-secret"},
		{name: "sample rate above one", key: "SENTRY_TRACES_SAMPLE_RATE", value: "9.5"},
		{name: "negative sample rate", key: "SENTRY_PROFILE_SESSION_SAMPLE_RATE", value: "-1"},
		{name: "port out of range", key: "DB_PORT", value: "70000"},
		{name: "empty database name", key: "DB_NAME", value: ""},
		{name: "negative hsts window", key: "SECURE_HSTS_SECONDS", value: "-5"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			sentryCfg := cfg
			sentryCfg.AddOns = sentryCfg.AddOns.With(config.Sentry)
			root, _ := writeTypedProject(t, sentryCfg, preset)
			overrideEnvValue(t, filepath.Join(root, envFile), tc.key, tc.value)

			out, ok := uvRunPydantic(t, root, `import sys; sys.path.insert(0, "."); import demo.conf`)
			if ok {
				t.Fatalf("%s=%q was accepted; the field is typed but not bounded:\n%s", tc.key, tc.value, out)
			}
			if !strings.Contains(out, tc.key) {
				t.Errorf("the failure does not name %s, so it is not actionable:\n%s", tc.key, out)
			}
		})
	}
}

func overrideEnvValue(t *testing.T, path, key, value string) {
	t.Helper()

	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}

	var kept []string
	for _, line := range strings.Split(string(contents), "\n") {
		if !strings.HasPrefix(line, key+"=") {
			kept = append(kept, line)
		}
	}
	kept = append(kept, key+"="+value)

	if err := os.WriteFile(path, []byte(strings.Join(kept, "\n")), 0o600); err != nil {
		t.Fatalf("rewriting %s: %v", path, err)
	}
}

func TestTypedSettingsReadTheGeneratedDatabaseCredentials(t *testing.T) {
	t.Parallel()

	cfg := typedProjectConfig()
	cfg.Database = config.PostgresDocker

	root, _ := writeTypedProject(t, cfg, config.DatabaseCredentials{Port: "5433", Password: "s3cret"})

	out, ok := uvRunPydantic(t, root, `
import sys
sys.path.insert(0, ".")
from demo.conf import env

assert env.DB_PORT == 5433, env.DB_PORT
assert env.DB_PASSWORD.get_secret_value() == "s3cret", env.DB_PASSWORD
assert "s3cret" not in repr(env), "the database password leaks through repr"
assert env.DB_USER == "django_user", env.DB_USER
print("OK")
`)

	if !ok {
		t.Fatalf("conf.py and the generated .env disagree about the database:\n%s", out)
	}
	if !strings.Contains(out, "OK") {
		t.Errorf("assertions did not run:\n%s", out)
	}
}
