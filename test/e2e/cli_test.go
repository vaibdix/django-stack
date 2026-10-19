package e2e

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

var binary string

func TestMain(m *testing.M) {
	os.Exit(runSuite(m))
}

func runSuite(m *testing.M) int {
	dir, err := os.MkdirTemp("", "django-stack-e2e")
	if err != nil {
		panic("creating temp dir: " + err.Error())
	}
	defer func() { _ = os.RemoveAll(dir) }()

	if _, err := os.Stat("../../go.mod"); err != nil {
		panic("e2e tests must run from within the module: " + err.Error())
	}

	binary = filepath.Join(dir, "django-stack")
	build := exec.Command("go", "build", "-o", binary, "../../cmd/django-stack")
	if out, err := build.CombinedOutput(); err != nil {
		panic("building the CLI under test: " + err.Error() + "\n" + string(out))
	}

	return m.Run()
}

func run(t *testing.T, args ...string) (stdout, stderr string, code int) {
	t.Helper()
	return runIn(t, "", args...)
}

func runIn(t *testing.T, dir string, args ...string) (stdout, stderr string, code int) {
	t.Helper()

	cmd := exec.Command(binary, args...)
	cmd.Dir = dir
	var outBuf, errBuf strings.Builder
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	err := cmd.Run()

	var exitErr *exec.ExitError
	switch {
	case err == nil:
		code = 0
	case errors.As(err, &exitErr):
		code = exitErr.ExitCode()
	default:
		t.Fatalf("running %v: %v", args, err)
	}
	return outBuf.String(), errBuf.String(), code
}

func TestVersionExitsZeroAndPrintsToStdout(t *testing.T) {
	t.Parallel()

	stdout, _, code := run(t, "-version")

	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout, "django-stack") {
		t.Errorf("version output does not name the program: %q", stdout)
	}
}

func TestHelpExitsZeroAndPrintsUsageToStderr(t *testing.T) {
	t.Parallel()

	_, stderr, code := run(t, "-h")

	if code != 0 {
		t.Errorf("exit code = %d for -h, want 0: asking for help is not an error", code)
	}
	for _, want := range []string{"Usage:", "-version", "-install"} {
		if !strings.Contains(stderr, want) {
			t.Errorf("usage output is missing %q:\n%s", want, stderr)
		}
	}
}

func TestUnknownFlagExitsNonZero(t *testing.T) {
	t.Parallel()

	_, stderr, code := run(t, "-nope")

	if code == 0 {
		t.Error("an unknown flag exited 0")
	}
	if !strings.Contains(stderr, "nope") {
		t.Errorf("stderr does not name the offending flag:\n%s", stderr)
	}
}

func TestStrayArgumentExitsNonZero(t *testing.T) {
	t.Parallel()

	_, stderr, code := run(t, "unexpected-arg")

	if code == 0 {
		t.Error("a stray positional argument exited 0")
	}
	if !strings.Contains(stderr, "unexpected-arg") {
		t.Errorf("stderr does not name the offending argument:\n%s", stderr)
	}
}

func TestTrailingArgumentAfterFlagExitsNonZero(t *testing.T) {
	t.Parallel()

	if _, _, code := run(t, "-version", "stray"); code == 0 {
		t.Error("a trailing argument after -version exited 0")
	}
}

func TestErrorsGoToStderrNotStdout(t *testing.T) {
	t.Parallel()

	stdout, stderr, _ := run(t, "-nope")

	if strings.TrimSpace(stdout) != "" {
		t.Errorf("error path wrote to stdout: %q", stdout)
	}
	if strings.TrimSpace(stderr) == "" {
		t.Error("error path wrote nothing to stderr")
	}
}
