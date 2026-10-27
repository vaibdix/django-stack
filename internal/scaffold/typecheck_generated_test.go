package scaffold

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/vaibdix/django-stack/internal/command/commandtest"
	"github.com/vaibdix/django-stack/internal/config"
	"github.com/vaibdix/django-stack/internal/port"
	"github.com/vaibdix/django-stack/internal/render"
	"github.com/vaibdix/django-stack/internal/report"
)

const mypyTimeout = 3 * time.Minute

func TestGeneratedSettingsSurviveMypy(t *testing.T) {
	t.Parallel()

	uv := requireUV(t)

	root := t.TempDir()
	writeTypedSettingsPackage(t, root)

	ctx, cancel := context.WithTimeout(context.Background(), mypyTimeout)
	t.Cleanup(cancel)

	cmd := exec.CommandContext(ctx, uv, "run", "--no-project", "--python", "3.13",
		"--with", "mypy", "--with", "pydantic-settings", "--with", "django==6.1.*", "--with", "django-stubs",
		"--with", "djangorestframework-stubs", "--with", "structlog", "--with", "django-structlog", "--with", "sentry-sdk",
		"mypy", "--config-file", render.PyprojectFile, "--no-error-summary", "--no-incremental", "proj")
	cmd.Dir = root

	out, runErr := cmd.CombinedOutput()
	if runErr != nil {
		t.Errorf("mypy rejects the settings this scaffold generates:\n%s", out)
	}
}

func writeTypedSettingsPackage(t *testing.T, root string) {
	t.Helper()

	cfg := config.Choices{
		Name:     "proj",
		App:      "catalog",
		Django:   "6.1",
		Database: config.SQLite,
		AddOns:   config.NewSet(config.RestFramework, config.MultiEnv, config.Pydantic, config.Structlog, config.Sentry),
	}

	e := &render.Project{Cfg: mustSpec(cfg)}
	settingsDir := filepath.Join(root, "proj", "settings")
	if err := os.MkdirAll(settingsDir, 0o755); err != nil {
		t.Fatalf("creating %s: %v", settingsDir, err)
	}

	if err := writeConfModule(e, root); err != nil {
		t.Fatalf("writing conf.py: %v", err)
	}

	settings := settingsWithObservability(t, cfg)
	data := render.MultiEnvSettingsData{AllAuth: true, Mailers: true, NativeCSP: true, Env: e.ProductionEnvExpressions()}

	if !strings.Contains(mypyConfigFor(t, cfg), "pydantic.mypy") {
		t.Fatal("applyMypy no longer enables the pydantic plugin for typed settings")
	}

	files := map[string]string{
		filepath.Join(root, render.PyprojectFile):   withoutDjangoPlugin(t, mypyConfigFor(t, cfg)),
		filepath.Join(root, "proj", "__init__.py"):  "",
		filepath.Join(settingsDir, "__init__.py"):   "",
		filepath.Join(settingsDir, "base.py"):       basePyFor(t, e, settings),
		filepath.Join(settingsDir, "local.py"):      render.Render("local.py.tmpl", data),
		filepath.Join(settingsDir, "production.py"): render.Render("production.py.tmpl", data),
		filepath.Join(settingsDir, "test.py"):       render.Render("test.py.tmpl", data),
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

func settingsWithRuntimeGenerics(t *testing.T, cfg config.Choices) string {
	t.Helper()
	cfg.AddOns = cfg.AddOns.With(config.Mypy)
	return settingsText((&render.Project{Cfg: mustSpec(cfg), Resolved: python313}))
}

func withoutDjangoPlugin(t *testing.T, pyproject string) string {
	t.Helper()
	const plugin = `"mypy_django_plugin.main", `
	if !strings.Contains(pyproject, plugin) || !strings.Contains(pyproject, "strict = true") {
		t.Fatalf("the generated mypy configuration changed shape:\n%s", pyproject)
	}
	pyproject = strings.Replace(pyproject, plugin, "", 1)
	return regexp.MustCompile(`(?s)\[tool\.django-stubs\]\n[^\[]*`).ReplaceAllString(pyproject, "")
}

func mypyConfigFor(t *testing.T, cfg config.Choices) string {
	t.Helper()

	root := t.TempDir()
	cfg.AddOns = cfg.AddOns.With(config.Mypy)
	e := &Env{Cfg: mustSpec(cfg), Resolved: python313, reporter: report.Nop{}, ports: port.Ports{Run: commandtest.NewFake()}}

	moduleDir := filepath.Join(root, cfg.ModuleName())
	if err := os.MkdirAll(moduleDir, 0o755); err != nil {
		t.Fatalf("creating the module directory: %v", err)
	}
	settings := filepath.Join(moduleDir, render.SettingsFile)
	if err := os.WriteFile(settings, []byte(djangoAdminSettingsFixture), 0o644); err != nil {
		t.Fatalf("writing the settings fixture: %v", err)
	}

	if err := writePyprojectWith(&e.Project, root, config.Mypy); err != nil {
		t.Fatalf("applyMypy: %v", err)
	}
	written, err := os.ReadFile(filepath.Join(root, render.PyprojectFile))
	if err != nil {
		t.Fatalf("reading pyproject.toml: %v", err)
	}
	return string(written)
}
