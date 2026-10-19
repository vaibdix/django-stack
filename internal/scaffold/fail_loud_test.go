package scaffold

import (
	"strings"
	"testing"

	"github.com/vaibdix/django-cli/internal/config"
	"github.com/vaibdix/django-cli/internal/scaffold/command/commandtest"
	"github.com/vaibdix/django-cli/internal/scaffold/pyenv"
)

// Each of these steps used to print a warning and carry on, leaving a project
// that looked finished but was missing its CSS, hooks or lock file.

func TestAFailedTailwindInstallFailsTheStep(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	fake := commandtest.NewFake()
	fake.FailWith("pnpm add --ignore-scripts tailwindcss @tailwindcss/cli", "ERR_PNPM_FETCH_404")
	e := &Env{Cfg: config.Project{ProjectName: "demo", SetupTailwind: true}, Runner: fake, Reporter: NopReporter{}, node: pnpmManager}

	err := e.applyTailwindStep(root)
	if err == nil || !strings.Contains(err.Error(), "install Tailwind CSS") || !strings.Contains(err.Error(), "ERR_PNPM_FETCH_404") {
		t.Fatalf("applyTailwindStep = %v, want the install failure and its cause", err)
	}
}

func TestTailwindRunsNodeWithTheManagerValidationFound(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	fake := commandtest.NewFake()
	e := &Env{Cfg: config.Project{ProjectName: "demo", SetupTailwind: true}, Runner: fake, Reporter: NopReporter{}, node: npmManager}

	if err := e.applyTailwindStep(root); err != nil {
		t.Fatalf("applyTailwindStep: %v", err)
	}
	if err := e.buildTailwindCSS(root); err != nil {
		t.Fatalf("buildTailwindCSS: %v", err)
	}
	for _, want := range []string{"npm install --ignore-scripts tailwindcss @tailwindcss/cli", "npm run build:css"} {
		if !fake.Ran(want) {
			t.Errorf("%q never ran; calls: %v", want, fake.CommandLines())
		}
	}
}

func TestAFailedPreCommitInstallFailsTheStep(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	fake := commandtest.NewFake()
	fake.FailWith(pyenv.InterpreterPath(root)+" -m pre_commit install", "not a git repository")
	e := &Env{Cfg: config.Project{ProjectName: "demo", SetupPreCommit: true, InitializeGit: true}, Runner: fake, Reporter: NopReporter{}}

	err := e.applyPreCommit(root)
	if err == nil || !strings.Contains(err.Error(), "pre-commit hooks") {
		t.Fatalf("applyPreCommit = %v, want the failed install reported", err)
	}
}

func TestAFailedUVLockFailsTheRun(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	fake := commandtest.NewFake()
	fake.FailWith("uv lock", "No solution found when resolving dependencies")
	e := &Env{Cfg: config.Project{ProjectName: "demo"}, Runner: fake, Reporter: NopReporter{}, toolchain: pyenv.UV}

	err := e.lockDependencies(root)
	if err == nil || !strings.Contains(err.Error(), LockFile) {
		t.Fatalf("lockDependencies = %v, want the missing lock file reported as a failure", err)
	}
}
