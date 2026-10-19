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
	fake.FailWith("pnpm add --ignore-scripts --save-exact tailwindcss@"+TailwindVersion+" @tailwindcss/cli@"+TailwindVersion, "ERR_PNPM_FETCH_404")
	e := &Env{Cfg: config.Project{ProjectName: "demo", SetupTailwind: true}, Runner: fake, Reporter: NopReporter{}, node: PNPM}

	err := e.applyTailwindStep(root)
	if err == nil || !strings.Contains(err.Error(), "install Tailwind CSS") || !strings.Contains(err.Error(), "ERR_PNPM_FETCH_404") {
		t.Fatalf("applyTailwindStep = %v, want the install failure and its cause", err)
	}
}

func TestTailwindRunsNodeWithTheManagerValidationFound(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	fake := commandtest.NewFake()
	e := &Env{Cfg: config.Project{ProjectName: "demo", SetupTailwind: true}, Runner: fake, Reporter: NopReporter{}, node: NPM}

	if err := e.applyTailwindStep(root); err != nil {
		t.Fatalf("applyTailwindStep: %v", err)
	}
	if err := e.buildTailwindCSS(root); err != nil {
		t.Fatalf("buildTailwindCSS: %v", err)
	}
	for _, want := range []string{"npm install --ignore-scripts --save-exact tailwindcss@" + TailwindVersion + " @tailwindcss/cli@" + TailwindVersion, "npm run build:css"} {
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

func TestADockerEnvFileWithoutChosenCredentialsIsRefused(t *testing.T) {
	t.Parallel()

	for _, db := range []config.Database{config.PostgresDocker, config.MySQLDocker} {
		e := &Env{Cfg: config.Project{ProjectName: "demo", DatabaseType: db}}
		if content, err := e.renderEnvFileContent(); err == nil {
			t.Errorf("%s: .env was rendered with credentials nobody chose:\n%s", db, content)
		}
	}
}

func TestAnUnparseableInterpreterVersionIsNotGuessed(t *testing.T) {
	t.Parallel()

	e := &Env{Cfg: config.Project{ProjectName: "demo", SelectedPythonVersion: "unknown"}}
	if err := e.writePythonVersionFile(t.TempDir()); err == nil {
		t.Error(".python-version was written from a guess instead of the selected interpreter")
	}
}

func TestMigrationsDoNotPretendToCreateTheProductionCacheTable(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	fake := commandtest.NewFake()
	rep := &RecordingReporter{}
	e := &Env{Cfg: config.Project{ProjectName: "demo"}, Runner: fake, Reporter: rep}

	if err := e.runMigrations(root); err != nil {
		t.Fatalf("runMigrations: %v", err)
	}
	for _, line := range fake.CommandLines() {
		if strings.Contains(line, "createcachetable") {
			t.Errorf("ran %q under the development settings, which configure no database cache", line)
		}
	}
	if strings.Contains(strings.Join(rep.Steps, "\n"), "cache table") {
		t.Errorf("reported a cache table that was never created: %q", rep.Steps)
	}
}
