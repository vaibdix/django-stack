package scaffold

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vaibdix/django-cli/internal/config"
)

const confRuntimeTimeout = 3 * time.Minute

func uvRunPydantic(t *testing.T, dir, script string) (string, bool) {
	t.Helper()

	uv, err := exec.LookPath("uv")
	if err != nil {
		t.Skip("uv is not installed")
	}

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

func writeTypedProject(t *testing.T, cfg config.Project) (root string, e *Env) {
	t.Helper()

	root = t.TempDir()
	e = &Env{Cfg: cfg, Reporter: NopReporter{}}

	if err := os.MkdirAll(filepath.Join(root, cfg.ModuleName()), 0o755); err != nil {
		t.Fatalf("creating the module directory: %v", err)
	}
	if err := e.writeConfModule(root); err != nil {
		t.Fatalf("writing conf.py: %v", err)
	}
	if err := e.writeEnvFile(root); err != nil {
		t.Fatalf("writing .env: %v", err)
	}
	return root, e
}

func typedProjectConfig() config.Project {
	return config.Project{
		ProjectName:       "demo",
		DjangoVersion:     "6.1",
		DatabaseType:      config.SQLite,
		SetupPydanticConf: true,
	}
}

func TestGeneratedSettingsParseTheDotEnvAsWritten(t *testing.T) {
	t.Parallel()

	root, _ := writeTypedProject(t, typedProjectConfig())

	out, ok := uvRunPydantic(t, root, `
import sys
sys.path.insert(0, '.')
from demo.conf import env

assert env.ALLOWED_HOSTS == ['localhost', '127.0.0.1'], env.ALLOWED_HOSTS
assert env.DEBUG is True, env.DEBUG
assert isinstance(env.SECRET_KEY, str) and env.SECRET_KEY, 'SECRET_KEY did not load'
print('OK')
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

	root, _ := writeTypedProject(t, typedProjectConfig())

	envPath := filepath.Join(root, EnvFile)
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
sys.path.insert(0, '.')
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
	cfg.DatabaseType = config.PostgresDocker
	cfg.PostgresDockerPassword = "pw"

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
			sentryCfg.SetupSentry = true
			root, _ := writeTypedProject(t, sentryCfg)
			overrideEnvValue(t, filepath.Join(root, EnvFile), tc.key, tc.value)

			out, ok := uvRunPydantic(t, root, "import sys; sys.path.insert(0, '.'); import demo.conf")
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
	cfg.DatabaseType = config.PostgresDocker
	cfg.PostgresDockerPort = "5433"
	cfg.PostgresDockerPassword = "s3cret"

	root, _ := writeTypedProject(t, cfg)

	out, ok := uvRunPydantic(t, root, `
import sys
sys.path.insert(0, '.')
from demo.conf import env

assert env.DB_PORT == 5433, env.DB_PORT
assert env.DB_PASSWORD == 's3cret', env.DB_PASSWORD
assert env.DB_USER == 'postgres', env.DB_USER
print('OK')
`)

	if !ok {
		t.Fatalf("conf.py and the generated .env disagree about the database:\n%s", out)
	}
	if !strings.Contains(out, "OK") {
		t.Errorf("assertions did not run:\n%s", out)
	}
}
