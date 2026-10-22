package scaffold

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vaibdix/django-stack/internal/config"
)

func writeEnvironSettings(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	cfg := config.Project{ProjectName: "demo", DjangoVersion: "6.1", DatabaseType: config.SQLite}
	e := &Env{Cfg: cfg, Reporter: NopReporter{}}

	moduleDir := filepath.Join(root, cfg.ModuleName())
	if err := os.MkdirAll(moduleDir, 0o755); err != nil {
		t.Fatalf("creating the module directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(moduleDir, "__init__.py"), nil, 0o644); err != nil {
		t.Fatalf("writing __init__.py: %v", err)
	}

	settingsPath := filepath.Join(moduleDir, SettingsFile)
	if err := os.WriteFile(settingsPath, []byte(djangoAdminSettingsFixture), 0o644); err != nil {
		t.Fatalf("writing the settings fixture: %v", err)
	}
	se, err := NewSettingsEditor(settingsPath)
	if err != nil {
		t.Fatalf("opening settings: %v", err)
	}
	if err := se.InjectEnvConfig(e.envConfig()); err != nil {
		t.Fatalf("InjectEnvConfig: %v", err)
	}
	if err := se.Save(); err != nil {
		t.Fatalf("saving settings: %v", err)
	}
	if err := e.writeEnvFile(root); err != nil {
		t.Fatalf("writing .env: %v", err)
	}
	return root
}

func debugUnderEnvironment(t *testing.T, root, environment string) string {
	t.Helper()

	uv := requireUV(t)

	ctx, cancel := context.WithTimeout(context.Background(), confRuntimeTimeout)
	t.Cleanup(cancel)

	cmd := exec.CommandContext(ctx, uv, "run", "--no-project", "--with", "django-environ", "--with", "django",
		"python", "-c", "import demo.settings as s; print('DEBUG=%s' % s.DEBUG)")
	cmd.Dir = root
	cmd.Env = environmentWithout(os.Environ(), "DEBUG", "ENVIRONMENT", "SECRET_KEY")
	if environment != "" {
		cmd.Env = append(cmd.Env, "ENVIRONMENT="+environment)
	}

	out, runErr := cmd.CombinedOutput()
	if runErr != nil {
		t.Fatalf("the generated settings do not import:\n%s", out)
	}
	return strings.TrimSpace(string(out))
}

func TestEnvironSettingsTakeDebugFromTheEnvironmentTheDotEnvSets(t *testing.T) {
	t.Parallel()

	root := writeEnvironSettings(t)

	if got := debugUnderEnvironment(t, root, ""); !strings.HasSuffix(got, "DEBUG=True") {
		t.Errorf("with the generated .env (ENVIRONMENT=local), settings report %q; "+
			"runserver would then redirect every request to https", got)
	}
	if got := debugUnderEnvironment(t, root, "production"); !strings.HasSuffix(got, "DEBUG=False") {
		t.Errorf("with ENVIRONMENT=production, settings report %q", got)
	}
}

func TestEnvironSettingsRefuseToStartWithoutASecretKey(t *testing.T) {
	t.Parallel()

	root := writeEnvironSettings(t)
	dotenvPath := filepath.Join(root, EnvFile)
	dotenv, err := os.ReadFile(dotenvPath)
	if err != nil {
		t.Fatalf("reading .env: %v", err)
	}
	var kept []string
	for _, line := range strings.Split(string(dotenv), "\n") {
		if !strings.HasPrefix(line, "SECRET_KEY=") {
			kept = append(kept, line)
		}
	}
	if err := os.WriteFile(dotenvPath, []byte(strings.Join(kept, "\n")), 0o600); err != nil {
		t.Fatalf("rewriting .env: %v", err)
	}

	uv := requireUV(t)
	ctx, cancel := context.WithTimeout(context.Background(), confRuntimeTimeout)
	t.Cleanup(cancel)

	cmd := exec.CommandContext(ctx, uv, "run", "--no-project", "--with", "django-environ", "--with", "django",
		"python", "-c", "import demo.settings")
	cmd.Dir = root
	cmd.Env = environmentWithout(os.Environ(), "SECRET_KEY", "ENVIRONMENT")
	out, runErr := cmd.CombinedOutput()
	if runErr == nil {
		t.Fatal("settings imported without a SECRET_KEY, so a deploy that forgets it runs on a public key")
	}
	if !strings.Contains(string(out), "SECRET_KEY") {
		t.Errorf("the failure does not name SECRET_KEY:\n%s", out)
	}
}
