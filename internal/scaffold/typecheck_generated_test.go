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

	"github.com/vaibdix/django-cli/internal/config"
	"github.com/vaibdix/django-cli/internal/scaffold/command/commandtest"
)

const mypyTimeout = 3 * time.Minute

func TestGeneratedSettingsSurviveMypy(t *testing.T) {
	t.Parallel()

	uv := requireUV(t)

	root := t.TempDir()
	writeTypedSettingsPackage(t, root)

	ctx, cancel := context.WithTimeout(context.Background(), mypyTimeout)
	t.Cleanup(cancel)

	// The same bar as a generated project's own mypy: strict, no explicit
	// Any, and every import resolved against real packages and stubs.
	cmd := exec.CommandContext(ctx, uv, "run", "--no-project", "--python", "3.13",
		"--with", "mypy", "--with", "pydantic-settings", "--with", "django==6.1.*", "--with", "django-stubs",
		"--with", "djangorestframework-stubs", "--with", "structlog", "--with", "django-structlog", "--with", "sentry-sdk",
		"mypy", "--config-file", PyprojectFile, "--no-error-summary", "--no-incremental", "proj")
	cmd.Dir = root

	out, runErr := cmd.CombinedOutput()
	if runErr != nil {
		t.Errorf("mypy rejects the settings this scaffold generates:\n%s", out)
	}
}

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
		SetupMultiEnv:      true,
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
	data := multiEnvSettingsData{AllAuth: true, Mailers: true, NativeCSP: true, Env: e.productionEnvExpressions()}

	if !strings.Contains(mypyConfigFor(t, cfg), "pydantic.mypy") {
		t.Fatal("applyMypy no longer enables the pydantic plugin for typed settings")
	}

	files := map[string]string{
		filepath.Join(root, PyprojectFile):          withoutDjangoPlugin(t, mypyConfigFor(t, cfg)),
		filepath.Join(root, "proj", "__init__.py"):  "",
		filepath.Join(settingsDir, "__init__.py"):   "",
		filepath.Join(settingsDir, "base.py"):       basePyFor(t, e, settings),
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

func settingsWithRuntimeGenerics(t *testing.T, cfg config.Project) string {
	t.Helper()

	root := t.TempDir()
	cfg.SetupMypy = true
	e := &Env{Cfg: cfg, Reporter: NopReporter{}, Runner: commandtest.NewFake()}

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

// withoutDjangoPlugin is the project's own generated mypy configuration
// (strict, no explicit Any, the settings re-export override) minus the Django
// plugin, which imports every INSTALLED_APPS entry and so needs the whole
// project. The real-generation test runs the project's full mypy.
func withoutDjangoPlugin(t *testing.T, pyproject string) string {
	t.Helper()
	const plugin = `"mypy_django_plugin.main", `
	if !strings.Contains(pyproject, plugin) || !strings.Contains(pyproject, "strict = true") {
		t.Fatalf("the generated mypy configuration changed shape:\n%s", pyproject)
	}
	pyproject = strings.Replace(pyproject, plugin, "", 1)
	return regexp.MustCompile(`(?s)\[tool\.django-stubs\]\n[^\[]*`).ReplaceAllString(pyproject, "")
}

func mypyConfigFor(t *testing.T, cfg config.Project) string {
	t.Helper()

	root := t.TempDir()
	cfg.SetupMypy = true
	e := &Env{Cfg: cfg, Reporter: NopReporter{}, Runner: commandtest.NewFake()}

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

func TestTheHTMXAppSurvivesStrictMypy(t *testing.T) {
	t.Parallel()

	uv := requireUV(t)

	root := t.TempDir()
	writeTypedHTMXApp(t, root)

	ctx, cancel := context.WithTimeout(context.Background(), mypyTimeout)
	t.Cleanup(cancel)

	cmd := exec.CommandContext(ctx, uv, "run", "--no-project",
		"--with", "mypy", "--with", "django", "--with", "django-stubs",
		"--with", "django-htmx", "--with", "pydantic", "--with", "pydantic-settings",
		"mypy", "--config-file", "mypy.ini", "--ignore-missing-imports",
		"--no-error-summary", "--no-incremental", "catalog")
	cmd.Dir = root

	out, runErr := cmd.CombinedOutput()
	if runErr != nil {
		t.Errorf("mypy rejects the htmx code this scaffold generates, which is the gap that let "+
			"request.htmx ship as an attr-defined error:\n%s", out)
	}
}

const htmxMypyConfig = `[mypy]
strict = True
disallow_any_explicit = True
follow_imports = silent
plugins = pydantic.mypy

[pydantic-mypy]
init_typed = True
init_forbid_extra = True
`

func writeTypedHTMXApp(t *testing.T, root string) {
	t.Helper()

	appDir := filepath.Join(root, "catalog")
	for _, dir := range []string{appDir, filepath.Join(root, "core")} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("creating %s: %v", dir, err)
		}
	}

	files := map[string]string{
		filepath.Join(root, "mypy.ini"):            htmxMypyConfig,
		filepath.Join(root, "core", "__init__.py"): "",
		filepath.Join(root, "core", "models.py"):   renderStatic("core_models.py.tmpl"),
		filepath.Join(appDir, "__init__.py"):       "",
		filepath.Join(appDir, "htmx.py"):           generateHtmxSupport(),
		filepath.Join(appDir, "fields.py"):         generateBookFields(true),
	}
	for path, content := range files {
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("writing %s: %v", path, err)
		}
	}

	htmx := files[filepath.Join(appDir, "htmx.py")]
	if !strings.Contains(htmx, "HtmxDetails") {
		t.Fatal("htmx.py does not reference django-htmx, so this test would prove nothing")
	}
}

func TestTheNinjaSchemasSurviveStrictMypy(t *testing.T) {
	t.Parallel()

	uv := requireUV(t)

	root := t.TempDir()
	apiDir := filepath.Join(root, "catalog", "api")
	if err := os.MkdirAll(apiDir, 0o755); err != nil {
		t.Fatalf("creating %s: %v", apiDir, err)
	}
	files := map[string]string{
		filepath.Join(root, "mypy.ini"):               htmxMypyConfig,
		filepath.Join(root, "catalog", "__init__.py"): "",
		filepath.Join(apiDir, "__init__.py"):          "",
		filepath.Join(apiDir, "schemas.py"):           generateNinjaPackage("catalog", "demo")["schemas.py"],
	}
	for path, content := range files {
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("writing %s: %v", path, err)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), mypyTimeout)
	t.Cleanup(cancel)

	cmd := exec.CommandContext(ctx, uv, "run", "--no-project",
		"--with", "mypy", "--with", "django", "--with", "django-stubs", "--with", "django-ninja",
		"mypy", "--config-file", "mypy.ini", "--ignore-missing-imports",
		"--no-error-summary", "--no-incremental", "catalog")
	cmd.Dir = root

	if out, runErr := cmd.CombinedOutput(); runErr != nil {
		t.Errorf("mypy rejects the Ninja schemas this scaffold generates:\n%s", out)
	}
}

func TestMissingImportsAreIgnoredOnlyForPackagesWithoutTypes(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		pydantic, allauth bool
		want              string
	}{
		{pydantic: false, allauth: false, want: `module = ["environ"]`},
		{pydantic: true, allauth: true, want: `module = ["allauth.*"]`},
		{pydantic: false, allauth: true, want: `module = ["environ", "allauth.*"]`},
		{pydantic: true, allauth: false, want: ""},
	} {
		e := &Env{Cfg: config.Project{SetupPydanticConf: tc.pydantic, SetupAllAuth: tc.allauth}}
		got := e.untypedImportsOverride()
		if (tc.want == "") != (got == "") || !strings.Contains(got, tc.want) {
			t.Errorf("pydantic=%v allauth=%v: override = %q, want one naming %q", tc.pydantic, tc.allauth, got, tc.want)
		}
	}
}
