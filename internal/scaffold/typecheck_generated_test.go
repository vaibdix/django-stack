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
	"github.com/vaibdix/django-cli/internal/scaffold/command"
)

const mypyTimeout = 3 * time.Minute

// The scaffold writes a [tool.mypy] section into every project that asks for
// it, but nothing checked that the code it generates alongside that config
// actually passes. It did not — a user met the errors on their first `mypy .`,
// in code they had not written.
//
// Scope: this runs mypy's own inference, not mypy_django_plugin. That plugin
// needs an importable settings module, which means installing and configuring
// the whole framework — too much for a unit test. So errors that only
// django-stubs can see are out of reach here and are covered by generating a
// real project instead.
func TestGeneratedSettingsSurviveMypy(t *testing.T) {
	t.Parallel()

	uv, err := exec.LookPath("uv")
	if err != nil {
		t.Skip("uv is not installed")
	}

	root := t.TempDir()
	writeTypedSettingsPackage(t, root)

	ctx, cancel := context.WithTimeout(context.Background(), mypyTimeout)
	t.Cleanup(cancel)

	// pydantic-settings is installed for real so env.EMAIL_PORT resolves to int
	// rather than Any — without it the MAILERS mismatch cannot be seen. Django
	// stays unresolved: mypy_django_plugin needs an importable settings module,
	// which would mean installing and configuring the whole framework, and the
	// errors this guards against are mypy's own inference either way.
	cmd := exec.CommandContext(ctx, uv, "run", "--no-project",
		"--with", "mypy", "--with", "pydantic-settings",
		"mypy", "--config-file", "mypy.ini", "--ignore-missing-imports",
		"--no-error-summary", "--no-incremental", "proj")
	cmd.Dir = root

	out, runErr := cmd.CombinedOutput()
	if runErr != nil {
		t.Errorf("mypy rejects the settings this scaffold generates:\n%s", out)
	}
}

// writeTypedSettingsPackage lays out the generated settings as a real package,
// so `from .base import *` resolves and env keeps the types conf.py gives it.
// Checking the rendered templates as loose files would report env as undefined
// and hide everything downstream of it.
func writeTypedSettingsPackage(t *testing.T, root string) {
	t.Helper()

	cfg := config.Project{
		ProjectName:        "proj",
		AppName:            "catalog",
		DjangoVersion:      "6.1",
		DatabaseType:       config.SQLite,
		SetupRestFramework: true,
		SetupStructlog:     true,
		SetupSentry:        true,
		SetupPydanticConf:  true,
	}

	e := &Env{Cfg: cfg, Reporter: NopReporter{}}
	settingsDir := filepath.Join(root, "proj", "settings")
	if err := os.MkdirAll(settingsDir, 0o755); err != nil {
		t.Fatalf("creating %s: %v", settingsDir, err)
	}

	if err := e.writeConfModule(root); err != nil {
		t.Fatalf("writing conf.py: %v", err)
	}

	settings := settingsWithObservability(t, cfg)
	data := multiEnvSettingsData{AllAuth: true, Mailers: true, Env: e.productionEnvExpressions()}

	// The pydantic plugin is what the generated pyproject.toml enables whenever
	// typed settings are on, and it is what teaches mypy that a BaseSettings
	// field with no default is filled from the environment rather than being a
	// required constructor argument. Checking without it would flag a line the
	// generated project is right about.
	if !strings.Contains(mypyConfigFor(t, cfg), "pydantic.mypy") {
		t.Fatal("applyMypy no longer enables the pydantic plugin for typed settings")
	}

	files := map[string]string{
		filepath.Join(root, "mypy.ini"):             "[mypy]\nplugins = pydantic.mypy\n",
		filepath.Join(root, "proj", "__init__.py"):  "",
		filepath.Join(settingsDir, "__init__.py"):   "",
		filepath.Join(settingsDir, "base.py"):       e.generateBasePy(settings),
		filepath.Join(settingsDir, "local.py"):      render("local.py.tmpl", data),
		filepath.Join(settingsDir, "production.py"): render("production.py.tmpl", data),
		filepath.Join(settingsDir, "test.py"):       render("test.py.tmpl", data),
	}
	for path, content := range files {
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("writing %s: %v", path, err)
		}
	}

	base := files[filepath.Join(settingsDir, "base.py")]
	if !strings.Contains(base, "from proj.conf import env") {
		t.Fatal("base.py lost the conf import, so env would type as Any and this test would prove nothing")
	}
	if !strings.Contains(base, "REST_FRAMEWORK") {
		t.Fatal("base.py has no REST block, so the dict this test exists to check is not being checked")
	}
}

// settingsWithRuntimeGenerics returns settings.py after applyMypy has run over
// it, which is where the django_stubs_ext patch is installed.
func settingsWithRuntimeGenerics(t *testing.T, cfg config.Project) string {
	t.Helper()

	root := t.TempDir()
	cfg.SetupMypy = true
	e := &Env{Cfg: cfg, Reporter: NopReporter{}, Runner: command.NewFake()}

	moduleDir := filepath.Join(root, cfg.ModuleName())
	if err := os.MkdirAll(moduleDir, 0o755); err != nil {
		t.Fatalf("creating the module directory: %v", err)
	}
	settingsPath := filepath.Join(moduleDir, SettingsFile)
	if err := os.WriteFile(settingsPath, []byte(djangoAdminSettingsFixture), 0o644); err != nil {
		t.Fatalf("writing the settings fixture: %v", err)
	}
	if err := e.applyMypy(root); err != nil {
		t.Fatalf("applyMypy: %v", err)
	}

	written, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("reading the generated settings: %v", err)
	}
	return string(written)
}

// mypyConfigFor returns the pyproject.toml applyMypy writes for this
// configuration, so callers read the real generator rather than a copy of it.
// applyMypy also edits settings.py — it installs the runtime-generics
// monkeypatch there — so the fixture has to exist first.
func mypyConfigFor(t *testing.T, cfg config.Project) string {
	t.Helper()

	root := t.TempDir()
	cfg.SetupMypy = true
	e := &Env{Cfg: cfg, Reporter: NopReporter{}, Runner: command.NewFake()}

	moduleDir := filepath.Join(root, cfg.ModuleName())
	if err := os.MkdirAll(moduleDir, 0o755); err != nil {
		t.Fatalf("creating the module directory: %v", err)
	}
	settings := filepath.Join(moduleDir, SettingsFile)
	if err := os.WriteFile(settings, []byte(djangoAdminSettingsFixture), 0o644); err != nil {
		t.Fatalf("writing the settings fixture: %v", err)
	}

	if err := e.applyMypy(root); err != nil {
		t.Fatalf("applyMypy: %v", err)
	}
	written, err := os.ReadFile(filepath.Join(root, PyprojectFile))
	if err != nil {
		t.Fatalf("reading pyproject.toml: %v", err)
	}
	return string(written)
}
