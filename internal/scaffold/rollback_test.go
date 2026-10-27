package scaffold

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vaibdix/django-stack/internal/command/commandtest"
	"github.com/vaibdix/django-stack/internal/config"
	"github.com/vaibdix/django-stack/internal/port"
	"github.com/vaibdix/django-stack/internal/pyenv"
)

func failingEnv(root string) *Env {
	return &Env{
		Cfg: mustSpec(config.Choices{
			Name:     "shop",
			Django:   "6.1.2",
			Database: config.SQLite,
		}),
		Resolved: config.Resolved{PythonPath: filepath.Join(root, "no-such-python")},
		root:     filepath.Join(root, "shop"),
		ports:    port.Ports{Run: commandtest.NewFake(), Clock: reviewClock},
	}
}

func TestAFailedRunRemovesTheDirectoryItCreated(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	e := failingEnv(root)

	err := e.Generate(context.Background())
	if err == nil {
		t.Fatal("Generate succeeded with an interpreter that does not exist")
	}
	if _, statErr := os.Stat(e.root); !os.IsNotExist(statErr) {
		t.Errorf("the partial project at %s was left behind, so a retry fails with "+
			"\"already exists and is not empty\"", e.root)
	}
	if !strings.Contains(err.Error(), "removed") {
		t.Errorf("the error does not say the partial project was removed: %v", err)
	}
}

func TestAFailedRunEmptiesButKeepsADirectoryThatAlreadyExisted(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	e := failingEnv(root)
	if err := os.Mkdir(e.root, 0o755); err != nil {
		t.Fatalf("creating the empty project directory: %v", err)
	}

	if err := e.Generate(context.Background()); err == nil {
		t.Fatal("Generate succeeded with an interpreter that does not exist")
	}
	entries, err := os.ReadDir(e.root)
	if err != nil {
		t.Fatalf("the directory the user created was deleted: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("the partial project was not cleaned out: %v", entries)
	}
}

func TestKeepOnFailureLeavesThePartialProjectForInspection(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	e := failingEnv(root)
	e.keepOnFailure = true

	err := e.Generate(context.Background())
	if err == nil {
		t.Fatal("Generate succeeded with an interpreter that does not exist")
	}
	if _, statErr := os.Stat(e.root); statErr != nil {
		t.Errorf("KeepOnFailure removed the partial project: %v", statErr)
	}
	if !strings.Contains(err.Error(), "kept at "+e.root) {
		t.Errorf("the error does not say where the partial project is: %v", err)
	}
}

func TestAProjectThatOnlyFailsVerificationIsKept(t *testing.T) {
	t.Parallel()

	undone := false
	e := &Env{}
	e.onRollback(func(context.Context) error { undone = true; return nil })

	cause := &VerificationError{Check: "test suite", Err: errors.New("exit status 1")}
	err := e.afterRun(context.Background(), cause)
	if !errors.Is(err, cause) && !errors.As(err, new(*VerificationError)) {
		t.Errorf("the verification error was replaced: %v", err)
	}
	if undone {
		t.Error("a generated project whose checks fail was rolled back; the user needs it to fix the check")
	}
}

func TestRollbackUndoesInReverseOrderAndReportsFailures(t *testing.T) {
	t.Parallel()

	var order []string
	e := &Env{}
	e.onRollback(func(context.Context) error { order = append(order, "directory"); return nil })
	e.onRollback(func(context.Context) error { order = append(order, "database"); return errors.New("docker gone") })

	err := e.rollback(context.Background(), errors.New("step failed"))
	if strings.Join(order, ",") != "database,directory" {
		t.Errorf("undo order = %v, want the database stopped before its directory is removed", order)
	}
	if !strings.Contains(err.Error(), "docker gone") || !strings.Contains(err.Error(), "step failed") {
		t.Errorf("the error hides the cause or the cleanup failure: %v", err)
	}
}

func TestACancelledRunSaysItWasCancelledAndCleansUp(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	e := failingEnv(root)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := e.Generate(ctx)
	if !errors.Is(err, context.Canceled) || !strings.Contains(err.Error(), "cancelled") {
		t.Errorf("a cancelled run reports %v", err)
	}
	if _, statErr := os.Stat(e.root); !os.IsNotExist(statErr) {
		t.Errorf("the cancelled run left %s behind", e.root)
	}
}

func TestTheEngineRefusesToRunWithoutAPython(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		cfg      config.Choices
		resolved config.Resolved
	}{
		"python": {cfg: config.Choices{Name: "shop", Django: "6.1.2"}},
	} {
		root := t.TempDir()
		e := &Env{Cfg: mustSpec(tc.cfg), Resolved: tc.resolved, root: filepath.Join(root, "shop"), ports: port.Ports{Run: commandtest.NewFake(), Clock: reviewClock}}
		if err := e.Generate(context.Background()); err == nil || !strings.Contains(err.Error(), "no ") {
			t.Errorf("missing %s: Generate = %v, want a refusal before anything is built", name, err)
		}
		if _, err := os.Stat(e.root); !os.IsNotExist(err) {
			t.Errorf("missing %s: a project directory was created", name)
		}
	}
}

func TestARunProbesForUVOnceAndRecordsTheChoice(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	e := failingEnv(root)
	fake, ok := e.ports.Run.(*commandtest.Fake)
	if !ok {
		t.Fatalf("failingEnv runner is %T", e.ports.Run)
	}
	fake.Responses["uv --version"] = []byte("uv 0.13.1\n")

	_ = e.Generate(context.Background())

	probes := 0
	for _, line := range fake.CommandLines() {
		if line == "uv --version" {
			probes++
		}
	}
	if probes != 1 {
		t.Errorf("uv was probed %d times in one run, want 1: the venv and every later step must agree", probes)
	}
	if e.Toolchain != pyenv.UV {
		t.Errorf("toolchain = %v, want uv", e.Toolchain)
	}
}

func TestARelativeRootIsResolvedAgainstTheWorkingDirectory(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	e := &Env{Cfg: mustSpec(config.Choices{Name: "demo"}), root: filepath.Join("nested", "target")}
	if err := e.prepareProjectDirectory(); err != nil {
		t.Fatalf("prepareProjectDirectory: %v", err)
	}
	wd, _ := os.Getwd()
	if want, got := filepath.Join(wd, "nested", "target"), e.target; got != want {
		t.Errorf("project directory = %q, want %q: a relative Root was replaced by the project name", got, want)
	}
}

func TestAUVRunIsBuiltBesideTheTargetAndMovedInOnlyAtTheEnd(t *testing.T) {
	t.Parallel()

	parent := t.TempDir()
	e := &Env{Cfg: mustSpec(config.Choices{Name: "shop"}), root: filepath.Join(parent, "shop")}
	e.Toolchain = pyenv.UV
	if err := e.prepareProjectDirectory(); err != nil {
		t.Fatalf("prepareProjectDirectory: %v", err)
	}
	if !e.staged() || filepath.Dir(e.dir) != parent || !strings.HasPrefix(filepath.Base(e.dir), ".shop.django-stack-") {
		t.Fatalf("dir = %s, want a hidden staging folder beside %s", e.dir, e.target)
	}
	if _, err := os.Stat(e.target); !os.IsNotExist(err) {
		t.Fatalf("the target exists before the project has been checked: %v", err)
	}
	for _, rel := range []string{"manage.py", filepath.Join("shop", "__pycache__", "settings.cpython-313.pyc"), filepath.Join("node_modules", "x", "__pycache__", "kept.pyc")} {
		if err := os.MkdirAll(filepath.Join(e.dir, filepath.Dir(rel)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(e.dir, rel), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	staging := e.dir
	if err := e.publish(); err != nil {
		t.Fatalf("publish: %v", err)
	}
	if e.dir != e.target {
		t.Errorf("after publishing, the run still writes to %s", e.dir)
	}
	if _, err := os.Stat(staging); !os.IsNotExist(err) {
		t.Errorf("the staging folder is still there: %v", err)
	}
	if _, err := os.Stat(filepath.Join(e.target, "manage.py")); err != nil {
		t.Errorf("the project did not move into place: %v", err)
	}
	if _, err := os.Stat(filepath.Join(e.target, "shop", "__pycache__")); !os.IsNotExist(err) {
		t.Errorf("compiled files that name the staging folder were published: %v", err)
	}
	if _, err := os.Stat(filepath.Join(e.target, "node_modules", "x", "__pycache__", "kept.pyc")); err != nil {
		t.Errorf("publishing walked into node_modules: %v", err)
	}
}

func TestAFailedUVRunLeavesNoStagingFolderBehind(t *testing.T) {
	t.Parallel()

	parent := t.TempDir()
	e := failingEnv(parent)
	fake, _ := e.ports.Run.(*commandtest.Fake)
	fake.Responses["uv --version"] = []byte("uv 0.13.1\n")

	if err := e.Generate(context.Background()); err == nil {
		t.Fatal("Generate succeeded with an interpreter that does not exist")
	}
	if entries, _ := os.ReadDir(parent); len(entries) != 0 {
		t.Errorf("a failed run left %v beside the target", entries)
	}
}

func TestKeepOnFailurePublishesAStagedPartialProject(t *testing.T) {
	t.Parallel()

	parent := t.TempDir()
	e := failingEnv(parent)
	e.keepOnFailure = true
	fake, _ := e.ports.Run.(*commandtest.Fake)
	fake.Responses["uv --version"] = []byte("uv 0.13.1\n")

	err := e.Generate(context.Background())
	if err == nil || !strings.Contains(err.Error(), "kept at "+e.root) {
		t.Fatalf("Generate = %v, want the partial project kept at %s", err, e.root)
	}
	entries, _ := os.ReadDir(parent)
	if len(entries) != 1 || entries[0].Name() != "shop" {
		t.Errorf("beside the target: %v, want only the kept project", entries)
	}
}

func TestAPipRunIsBuiltInPlace(t *testing.T) {
	t.Parallel()

	e := &Env{Cfg: mustSpec(config.Choices{Name: "shop"}), root: filepath.Join(t.TempDir(), "shop")}
	e.Toolchain = pyenv.Pip
	if err := e.prepareProjectDirectory(); err != nil {
		t.Fatalf("prepareProjectDirectory: %v", err)
	}
	if e.staged() || e.dir != e.target {
		t.Errorf("a pip run is staged in %s; pip environments record their own path", e.dir)
	}
}
