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
	"github.com/vaibdix/django-stack/internal/livetest"
	"github.com/vaibdix/django-stack/internal/report"
)

const ruffTimeout = 2 * time.Minute

func ruffCommand(t *testing.T, args ...string) (*exec.Cmd, bool) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), ruffTimeout)
	t.Cleanup(cancel)

	if path, err := exec.LookPath("uvx"); err == nil && !testing.Short() {
		return exec.CommandContext(ctx, path, append([]string{"ruff@" + ruffVersion}, args...)...), true
	}
	if path, err := exec.LookPath("ruff"); err == nil && localRuffIsPinned(ctx, path) {
		return exec.CommandContext(ctx, path, args...), true
	}
	return nil, false
}

const ruffVersion = "0.15.0"

func localRuffIsPinned(ctx context.Context, path string) bool {
	out, err := exec.CommandContext(ctx, path, "--version").Output()
	return err == nil && strings.TrimSpace(string(out)) == "ruff "+ruffVersion
}

func writeGeneratedRuffConfig(t *testing.T, root, app string) {
	t.Helper()

	e := &Env{
		cfg: mustSpec(config.Choices{
			Name:   "lintcheck",
			App:    app,
			Django: "6.1",
		}),
		resolved: config.Resolved{PythonVersion: "Python 3.13.2"},
		reporter: report.Nop{},
	}

	e.dir = root
	if err := e.writePyprojectWith(config.Ruff); err != nil {
		t.Fatalf("generating the ruff configuration: %v", err)
	}

	written, err := os.ReadFile(filepath.Join(root, pyprojectFile))
	if err != nil {
		t.Fatalf("reading the generated %s: %v", pyprojectFile, err)
	}
	if !strings.Contains(string(written), "extend-select") {
		t.Fatalf("the generated config selects no rules, so this test would lint against ruff's defaults:\n%s", written)
	}
}

func TestGeneratedPythonSurvivesRuff(t *testing.T) {
	t.Parallel()

	cmd, ok := ruffCommand(t, "--version")
	if !ok {
		livetest.MissingTool(t, "neither ruff nor uvx is installed")
	}
	if err := cmd.Run(); err != nil {
		livetest.MissingTool(t, "ruff is not runnable: %v", err)
	}

	root := t.TempDir()

	const app = "catalog"
	writeGeneratedRuffConfig(t, root, app)
	appDir := filepath.Join(root, app)
	apiDir := filepath.Join(appDir, "api")
	commandsDir := filepath.Join(appDir, "management", "commands")
	settingsDir := filepath.Join(root, "proj", "settings")
	for _, d := range []string{appDir, apiDir, commandsDir, settingsDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatalf("creating %s: %v", d, err)
		}
	}

	files := map[string]string{
		filepath.Join(appDir, "forms.py"):                   generateBookForms(bookFormsData{Tailwind: true, RuntimeGenerics: true}),
		filepath.Join(appDir, "models.py"):                  generateBookModel("catalog", true),
		filepath.Join(apiDir, "__init__.py"):                generateAPIPackage("catalog")["__init__.py"],
		filepath.Join(apiDir, "serializers.py"):             generateAPIPackage("catalog")["serializers.py"],
		filepath.Join(apiDir, "views.py"):                   generateAPIPackage("catalog")["views.py"],
		filepath.Join(apiDir, "urls.py"):                    generateAPIPackage("catalog")["urls.py"],
		filepath.Join(appDir, "admin.py"):                   generateAdminPy(typingData{RuntimeGenerics: true}),
		filepath.Join(appDir, "views.py"):                   renderBookViews(bookViewsData{AppName: app, HTMX: true, RuntimeGenerics: true}),
		filepath.Join(appDir, "urls.py"):                    generateAppURLs(app),
		filepath.Join(commandsDir, "create_sample_data.py"): generateSampleData(app, config.SQLite),
		filepath.Join(appDir, "fields.py"):                  generateBookFields(typingData{RuntimeGenerics: true}),
		filepath.Join(appDir, "tests.py"):                   render("tests_book.py.tmpl", testsData{AppName: app, AppNameTitle: "Catalog", Ruff: true}),
		filepath.Join(appDir, "test_models.py"):             render("tests_model.py.tmpl", appNames{AppName: app, AppNameTitle: "Catalog"}),

		filepath.Join(settingsDir, "__init__.py"):   "",
		filepath.Join(settingsDir, "local.py"):      render("local.py.tmpl", environSettingsData()),
		filepath.Join(settingsDir, "production.py"): render("production.py.tmpl", environSettingsData()),
		filepath.Join(settingsDir, "test.py"):       render("test.py.tmpl", environSettingsData()),

		filepath.Join(root, "proj", "conf.py"): render("conf.py.tmpl", pydanticConfData()),

		filepath.Join(settingsDir, "base.py"): settingsWithObservability(t, config.Choices{
			Name:     "proj",
			Django:   "6.1",
			Database: config.SQLite,
			AddOns:   config.NewSet(config.Ruff, config.Pydantic, config.Structlog, config.Sentry),
		}),
	}
	for path, content := range files {
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("writing %s: %v", path, err)
		}
	}

	format, _ := ruffCommand(t, "format", "--check", "--no-cache", ".")
	format.Dir = root
	if out, err := format.CombinedOutput(); err != nil {
		t.Errorf("the generated Python is not already formatted, and nothing formats it after generation:\n%s", out)
	}

	check, _ := ruffCommand(t, "check", "--no-cache", "--no-fix", ".")
	check.Dir = root
	if out, err := check.CombinedOutput(); err != nil {
		t.Errorf("ruff rejects the generated Python, which fails the project's own check:\n%s", out)
	}
}

func TestGeneratedPythonRespectsTheGeneratedLineLimit(t *testing.T) {
	t.Parallel()

	const limit = 120

	sources := map[string]string{
		"forms.py (tailwind)":         generateBookForms(bookFormsData{Tailwind: true, RuntimeGenerics: true}),
		"forms.py (plain)":            generateBookForms(bookFormsData{Tailwind: false, RuntimeGenerics: true}),
		"views.py (htmx)":             renderBookViews(bookViewsData{AppName: "catalog", HTMX: true, RuntimeGenerics: true}),
		"models.py":                   generateBookModel("catalog", true),
		"admin.py":                    generateAdminPy(typingData{RuntimeGenerics: true}),
		"sample_data.py":              generateSampleData("catalog", config.SQLite),
		"settings/base":               render("local.py.tmpl", environSettingsData()),
		"settings/production":         render("production.py.tmpl", environSettingsData()),
		"settings/production (typed)": render("production.py.tmpl", pydanticSettingsData()),
		"conf.py":                     render("conf.py.tmpl", pydanticConfData()),
		"fields.py":                   generateBookFields(typingData{RuntimeGenerics: true}),
		"ninja/api.py":                generateNinjaPackage("catalog", "my-shop")["api.py"],
		"ninja/router.py":             generateNinjaPackage("catalog", "my-shop")["router.py"],
		"ninja/schemas.py":            generateNinjaPackage("catalog", "my-shop")["schemas.py"],
	}

	for name, src := range sources {
		for i, line := range strings.Split(src, "\n") {
			if len(line) > limit {
				t.Errorf("%s line %d is %d characters, over the generated limit of %d:\n  %s",
					name, i+1, len(line), limit, line)
			}
		}
	}
}

func TestWithoutRuffNoGeneratedFileCarriesASuppression(t *testing.T) {
	t.Parallel()

	for _, ruff := range []bool{false, true} {
		cfg := config.Choices{
			Name: "my-shop", App: "catalog", Database: config.SQLite, Stack: config.StackHTMX,
			AddOns: addOnsWhere(map[config.AddOn]bool{config.RestFramework: true, config.MultiEnv: true, config.Ruff: ruff}),
		}
		e := &Env{cfg: mustSpec(cfg)}
		sources := map[string]string{
			"tests_book.py": render("tests_book.py.tmpl", e.testsData()),
			"tests_api.py":  e.apiTestsContent(),
			"conf.py":       render("conf.py.tmpl", e.confTemplateData()),
			"settings":      e.singleFileEnvironmentBlock(),
		}
		for name, source := range sources {
			if has := strings.Contains(source, "# noqa"); has != ruff {
				t.Errorf("ruff=%t: %s carries a noqa = %t:\n%s", ruff, name, has, source)
			}
		}
	}
}
