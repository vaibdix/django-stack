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

	if path, err := exec.LookPath("uvx"); err == nil {
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
	settingsDir := filepath.Join(root, "proj", "settings")
	for _, d := range []string{appDir, settingsDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatalf("creating %s: %v", d, err)
		}
	}

	files := map[string]string{
		filepath.Join(appDir, "forms.py"):       generateBookForms(true),
		filepath.Join(appDir, "models.py"):      generateBookModel(),
		filepath.Join(appDir, "serializers.py"): generateBookSerializer(),
		filepath.Join(appDir, "api.py"):         generateBookAPI(),
		filepath.Join(appDir, "admin.py"):       generateAdminPy(),
		filepath.Join(appDir, "views.py"):       generateBookViews(app, true),
		filepath.Join(appDir, "urls.py"):        generateAppURLs(app, true),
		filepath.Join(appDir, "sample_data.py"): generateSampleData(app),
		filepath.Join(appDir, "tests.py"):       render("tests_book.py.tmpl", appNames{AppName: app, AppNameTitle: "Catalog"}),

		filepath.Join(settingsDir, "__init__.py"):   "",
		filepath.Join(settingsDir, "local.py"):      render("local.py.tmpl", struct{ AllAuth, Mailers bool }{true, true}),
		filepath.Join(settingsDir, "production.py"): render("production.py.tmpl", struct{ AllAuth, Mailers bool }{true, true}),
		filepath.Join(settingsDir, "test.py"):       render("test.py.tmpl", struct{ AllAuth, Mailers bool }{true, true}),
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
		"forms.py (tailwind)": generateBookForms(true),
		"forms.py (plain)":    generateBookForms(false),
		"views.py (htmx)":     generateBookViews("catalog", true),
		"models.py":           generateBookModel(),
		"admin.py":            generateAdminPy(),
		"sample_data.py":      generateSampleData("catalog"),
		"settings/base":       render("local.py.tmpl", struct{ AllAuth, Mailers bool }{true, true}),
		"settings/production": render("production.py.tmpl", struct{ AllAuth, Mailers bool }{true, true}),
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
