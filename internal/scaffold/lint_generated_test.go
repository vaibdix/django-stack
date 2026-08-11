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

const generatedRuffConfig = `[project]
name = "lintcheck"
version = "0.1.0"
requires-python = ">=3.11"

[tool.ruff]
target-version = "py311"
line-length = 120

[tool.ruff.lint]
select = ["A","B","C4","DJ","E","F","I","S","SIM","UP","W"]
ignore = ["RUF012", "S101", "SIM102"]

[tool.ruff.lint.per-file-ignores]
"*/settings/*.py" = ["F403", "F405"]
`

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
	if err := os.WriteFile(filepath.Join(root, "pyproject.toml"), []byte(generatedRuffConfig), 0o644); err != nil {
		t.Fatalf("writing pyproject.toml: %v", err)
	}

	const app = "catalog"
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

		filepath.Join(settingsDir, "__init__.py"):   "",
		filepath.Join(settingsDir, "local.py"):      renderStatic("local.py.tmpl"),
		filepath.Join(settingsDir, "production.py"): renderStatic("production.py.tmpl"),
		filepath.Join(settingsDir, "test.py"):       renderStatic("test.py.tmpl"),
	}
	for path, content := range files {
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("writing %s: %v", path, err)
		}
	}

	fix, _ := ruffCommand(t, "check", "--no-cache", "--fix", ".")
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
		"settings/base":       renderStatic("local.py.tmpl"),
		"settings/production": renderStatic("production.py.tmpl"),
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
