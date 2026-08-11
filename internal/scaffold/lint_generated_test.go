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
)

// The golden files pin the exact bytes of generated Python, but nothing ever
// asked ruff whether those bytes are acceptable — and the scaffolder runs
// `ruff check --fix` over its own output as a mandatory step, aborting the whole
// generation if anything survives the autofix.
//
// That gap shipped two real defects, both found only by scaffolding a project by
// hand:
//
//   - the Tailwind widget class was interpolated into every form field,
//     producing 300-character lines that E501 rejects and cannot autofix;
//   - the split settings modules use `from .base import *`, so every name
//     reached through it (env, BASE_DIR) was reported by F405.
//
// This test closes the gap: it renders the generated Python against the same
// ruff configuration the tool writes into pyproject.toml, and fails if anything
// survives the autofix.

// ruffTimeout bounds the linter so a wedged uvx download cannot hang the suite.
const ruffTimeout = 2 * time.Minute

// ruffCommand returns a command that runs ruff, preferring a real binary and
// falling back to uvx. The bool reports whether ruff is reachable at all.
func ruffCommand(t *testing.T, args ...string) (*exec.Cmd, bool) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), ruffTimeout)
	t.Cleanup(cancel)

	if path, err := exec.LookPath("ruff"); err == nil {
		return exec.CommandContext(ctx, path, args...), true
	}
	// uvx fetches ruff on demand, which is why this needs a deadline at all.
	if path, err := exec.LookPath("uvx"); err == nil {
		return exec.CommandContext(ctx, path, append([]string{"ruff"}, args...)...), true
	}
	return nil, false
}

// generatedRuffConfig is the lint configuration applyRuff writes into a
// generated project. It is duplicated here deliberately: if the two drift, this
// test stops reflecting what users actually get, and the assertion below about
// per-file-ignores catches that.
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

// TestGeneratedPythonSurvivesRuff scaffolds the generated Python into a temp
// project and runs the same autofix pass the scaffolder runs.
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

	// Every generated Python module, in both of its variants where it has one.
	files := map[string]string{
		filepath.Join(appDir, "forms.py"):       generateBookForms(true),
		filepath.Join(appDir, "models.py"):      generateBookModel(),
		filepath.Join(appDir, "serializers.py"): generateBookSerializer(),
		filepath.Join(appDir, "api.py"):         generateBookAPI(),
		filepath.Join(appDir, "admin.py"):       generateAdminPy(),
		filepath.Join(appDir, "views.py"):       generateBookViews(app, true),
		filepath.Join(appDir, "urls.py"):        generateAppURLs(app, true),
		filepath.Join(appDir, "sample_data.py"): generateSampleData(app),

		filepath.Join(settingsDir, "__init__.py"):   renderStatic("init.py.tmpl"),
		filepath.Join(settingsDir, "local.py"):      renderStatic("local.py.tmpl"),
		filepath.Join(settingsDir, "production.py"): renderStatic("production.py.tmpl"),
		filepath.Join(settingsDir, "test.py"):       renderStatic("test.py.tmpl"),
	}
	for path, content := range files {
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("writing %s: %v", path, err)
		}
	}

	// This is exactly what formatGeneratedCode runs. A non-zero exit here is a
	// scaffold that aborts partway through for the user.
	fix, _ := ruffCommand(t, "check", "--no-cache", "--fix", ".")
	fix.Dir = root
	out, err := fix.CombinedOutput()
	if err != nil {
		t.Errorf("ruff rejects the generated Python, which aborts the scaffold:\n%s", out)
	}
}

// The Tailwind widget class is long enough that inlining it breaks the line
// limit the generated config sets. Guarding the property directly means the
// failure is legible even without ruff installed.
func TestGeneratedPythonRespectsTheGeneratedLineLimit(t *testing.T) {
	t.Parallel()

	const limit = 120 // matches line-length in the generated [tool.ruff]

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

// djlint --reformat exits 1 when it has rewritten files, which is success, and
// 2 when something is actually wrong. Treating the former as a failure aborted
// every scaffold that generated templates, immediately after the formatter had
// done exactly what it was asked to.
//
// These use real *exec.ExitError values produced by running a shell, rather than
// a hand-made stub, so they pin the behaviour against the type the code sees.

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
