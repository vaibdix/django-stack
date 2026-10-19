package scaffold

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/vaibdix/django-cli/internal/config"
	"github.com/vaibdix/django-cli/internal/scaffold/command"
)

const ruffTimeout = 2 * time.Minute

func ruffCommand(t *testing.T, args ...string) (*exec.Cmd, bool) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), ruffTimeout)
	t.Cleanup(cancel)

	if path, err := exec.LookPath("ruff"); err == nil {
		return exec.CommandContext(ctx, path, args...), true
	}

	if path, err := exec.LookPath("uvx"); err == nil && !testing.Short() {
		return exec.CommandContext(ctx, path, append([]string{"ruff"}, args...)...), true
	}
	return nil, false
}

func writeGeneratedRuffConfig(t *testing.T, root, app string) {
	t.Helper()

	e := &Env{
		Cfg: config.Project{
			ProjectName:           "lintcheck",
			AppName:               app,
			DjangoVersion:         "6.1",
			SelectedPythonVersion: "Python 3.13.2",
		},
		Reporter: NopReporter{},
	}

	if err := e.applyRuff(root); err != nil {
		t.Fatalf("generating the ruff configuration: %v", err)
	}

	written, err := os.ReadFile(filepath.Join(root, PyprojectFile))
	if err != nil {
		t.Fatalf("reading the generated %s: %v", PyprojectFile, err)
	}
	if !strings.Contains(string(written), "extend-select") {
		t.Fatalf("the generated config selects no rules, so this test would lint against ruff's defaults:\n%s", written)
	}
}

func TestGeneratedPythonSurvivesRuff(t *testing.T) {
	t.Parallel()

	cmd, ok := ruffCommand(t, "--version")
	if !ok {
		t.Skip("neither ruff nor uvx is installed")
	}
	if err := cmd.Run(); err != nil {
		t.Skipf("ruff is not runnable: %v", err)
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
		filepath.Join(appDir, "forms.py"):                   generateBookForms(true, true),
		filepath.Join(appDir, "models.py"):                  generateBookModel("catalog", true, "6.1", config.SQLite),
		filepath.Join(apiDir, "__init__.py"):                generateAPIPackage("catalog")["__init__.py"],
		filepath.Join(apiDir, "serializers.py"):             generateBookSerializer("catalog"),
		filepath.Join(apiDir, "views.py"):                   generateAPIPackage("catalog")["views.py"],
		filepath.Join(apiDir, "urls.py"):                    generateAPIPackage("catalog")["urls.py"],
		filepath.Join(appDir, "admin.py"):                   generateAdminPy(true),
		filepath.Join(appDir, "views.py"):                   generateBookViews(app, true, true),
		filepath.Join(appDir, "urls.py"):                    generateAppURLs(app),
		filepath.Join(commandsDir, "create_sample_data.py"): generateSampleData(app, config.SQLite),
		filepath.Join(appDir, "fields.py"):                  generateBookFields(true),
		filepath.Join(appDir, "tests.py"):                   render("tests_book.py.tmpl", testsData{appNames: appNames{AppName: app, AppNameTitle: "Catalog"}}),
		filepath.Join(appDir, "test_models.py"):             render("tests_model.py.tmpl", appNames{AppName: app, AppNameTitle: "Catalog"}),

		filepath.Join(settingsDir, "__init__.py"):   "",
		filepath.Join(settingsDir, "local.py"):      render("local.py.tmpl", environSettingsData()),
		filepath.Join(settingsDir, "production.py"): render("production.py.tmpl", environSettingsData()),
		filepath.Join(settingsDir, "test.py"):       render("test.py.tmpl", environSettingsData()),

		filepath.Join(root, "proj", "conf.py"): render("conf.py.tmpl", pydanticConfData()),

		filepath.Join(settingsDir, "base.py"): settingsWithObservability(t, config.Project{
			ProjectName:       "proj",
			DjangoVersion:     "6.1",
			DatabaseType:      config.SQLite,
			SetupStructlog:    true,
			SetupSentry:       true,
			SetupPydanticConf: true,
		}),
	}
	for path, content := range files {
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("writing %s: %v", path, err)
		}
	}

	fix, _ := ruffCommand(t, "check", "--no-cache", "--no-fix", ".")
	fix.Dir = root
	out, err := fix.CombinedOutput()
	if err != nil {
		t.Errorf("ruff rejects the generated Python, which aborts the scaffold:\n%s", out)
	}
}

func TestGeneratedPythonRespectsTheGeneratedLineLimit(t *testing.T) {
	t.Parallel()

	const limit = 120

	sources := map[string]string{
		"forms.py (tailwind)":         generateBookForms(true, true),
		"forms.py (plain)":            generateBookForms(false, true),
		"views.py (htmx)":             generateBookViews("catalog", true, true),
		"models.py":                   generateBookModel("catalog", true, "6.1", config.SQLite),
		"admin.py":                    generateAdminPy(true),
		"sample_data.py":              generateSampleData("catalog", config.SQLite),
		"settings/base":               render("local.py.tmpl", environSettingsData()),
		"settings/production":         render("production.py.tmpl", environSettingsData()),
		"settings/production (typed)": render("production.py.tmpl", pydanticSettingsData()),
		"conf.py":                     render("conf.py.tmpl", pydanticConfData()),
		"fields.py":                   generateBookFields(true),
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

func exitErrorWithCode(t *testing.T, code int) error {
	t.Helper()

	err := exec.Command("sh", "-c", "exit "+strconv.Itoa(code)).Run() //nolint:noctx // trivial, local
	if err == nil {
		t.Fatalf("expected a non-zero exit for code %d", code)
	}
	return err
}

func TestFormattingTreatsDjlintReformatAsSuccess(t *testing.T) {
	t.Parallel()

	if got := djlintDidReformat(exitErrorWithCode(t, 1)); !got {
		t.Error("exit 1 from djlint --reformat means files were rewritten; that is success")
	}
}

func TestFormattingStillFailsOnRealDjlintErrors(t *testing.T) {
	t.Parallel()

	for _, code := range []int{2, 3, 127} {
		if djlintDidReformat(exitErrorWithCode(t, code)) {
			t.Errorf("exit %d is a genuine djlint failure and must still abort the scaffold", code)
		}
	}

	if djlintDidReformat(errors.New("some other failure")) {
		t.Error("a non-exit error must not be mistaken for a successful reformat")
	}
}

func TestDjlintReformatIsRecognisedThroughTheRunner(t *testing.T) {
	t.Parallel()

	wrapped := &command.Error{
		Name:   "python",
		Args:   []string{"-m", "djlint", "--reformat"},
		Output: []byte("3 files reformatted"),
		Err:    exitErrorWithCode(t, djlintReformattedExitCode),
	}

	if !djlintDidReformat(wrapped) {
		t.Error("a reformat exit wrapped in command.Error was read as a failure; " +
			"the scaffold would abort right after djlint did its job correctly")
	}

	genuine := &command.Error{
		Name: "python",
		Args: []string{"-m", "djlint", "--reformat"},
		Err:  exitErrorWithCode(t, 2),
	}
	if djlintDidReformat(genuine) {
		t.Error("a wrapped exit 2 is a genuine djlint failure and must still abort the scaffold")
	}
}

func TestLintSuppressionsAreDroppedWithoutRuff(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	source := "TEST_PASSWORD = 'x'  # noqa: S105\nfrom .base import *  # noqa: F401,F403\n"
	path := filepath.Join(root, "tests.py")
	if err := os.WriteFile(path, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := (&Env{}).dropLintSuppressions(root); err != nil {
		t.Fatalf("dropLintSuppressions: %v", err)
	}
	got, _ := os.ReadFile(path)
	if strings.Contains(string(got), "noqa: S105") {
		t.Errorf("the S105 suppression survived in a project without Ruff:\n%s", got)
	}
	if strings.Contains(string(got), "noqa") {
		t.Errorf("a suppression survived in a project with no linter configured:\n%s", got)
	}
	if !strings.Contains(string(got), "from .base import *\n") {
		t.Errorf("the code itself was changed, not just the comment:\n%s", got)
	}
}
