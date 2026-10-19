package scaffold

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vaibdix/django-cli/internal/config"
	"github.com/vaibdix/django-cli/internal/scaffold/command/commandtest"
	"github.com/vaibdix/django-cli/internal/scaffold/pyenv"
)

func closureCall(root string, specs ...string) string {
	return strings.Join(append([]string{pyenv.InterpreterPath(root), "-c", dependencyClosureScript}, specs...), " ")
}

func TestRequirementsKeepDevToolsOutOfTheRuntimeFile(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	fake := commandtest.NewFake()
	fake.Responses[closureCall(root, "django==6.1.2", "whitenoise")] = []byte("asgiref==3.12.1\ndjango==6.1.2\nsqlparse==0.6.0\nwhitenoise==6.12.0\n")
	fake.Responses[closureCall(root, "ruff", "mypy")] = []byte("mypy==2.4.0\nmypy-extensions==1.1.0\nruff==0.17.1\n")

	e := &Env{Cfg: config.Project{ProjectName: "demo"}, Runner: fake, Reporter: NopReporter{}, hashLookup: fakeHashes}
	e.deps.addRuntime("django==6.1.2", "whitenoise")
	e.deps.addDev("ruff", "mypy")

	if err := e.writeRequirementsFiles(root); err != nil {
		t.Fatalf("writeRequirementsFiles: %v", err)
	}

	runtime, _ := os.ReadFile(filepath.Join(root, RequirementsFile))
	dev, _ := os.ReadFile(filepath.Join(root, DevRequirementsFile))
	if !strings.Contains(string(runtime), "sqlparse==0.6.0 \\\n    --hash=sha256:sqlparse\n") {
		t.Errorf("requirements.txt does not pin the transitive dependencies:\n%s", runtime)
	}
	for _, tool := range []string{"ruff", "mypy"} {
		if strings.Contains(string(runtime), tool) {
			t.Errorf("requirements.txt, which production installs, carries %s:\n%s", tool, runtime)
		}
	}
	if !strings.HasPrefix(string(dev), "-r requirements.txt\n") || !strings.Contains(string(dev), "ruff==0.17.1 \\\n    --hash=sha256:ruff") {
		t.Errorf("requirements-dev.txt does not layer the tools on the runtime set:\n%s", dev)
	}
}

func TestAnUnreadableDependencyTreeFailsTheRun(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	fake := commandtest.NewFake()
	fake.FailWithPrefix(pyenv.InterpreterPath(root)+" -c", "no pip in this venv")

	e := &Env{Cfg: config.Project{ProjectName: "demo"}, Runner: fake, Reporter: NopReporter{}, hashLookup: fakeHashes}
	e.deps.addRuntime("django==6.1.2")
	e.deps.addDev("ruff")

	err := e.writeRequirementsFiles(root)
	if err == nil || !strings.Contains(err.Error(), "dependency tree") {
		t.Fatalf("writeRequirementsFiles = %v, want the failed resolve reported", err)
	}
	if _, statErr := os.Stat(filepath.Join(root, RequirementsFile)); !os.IsNotExist(statErr) {
		t.Errorf("a partial %s was written after the resolve failed", RequirementsFile)
	}
}

func fakeHashes(_ context.Context, project, _ string) ([]string, error) {
	return []string{"sha256:" + project}, nil
}

func TestAFailedHashLookupFailsTheRunInsteadOfDroppingTheHashes(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	fake := commandtest.NewFake()
	fake.Responses[closureCall(root, "django==6.1.2")] = []byte("asgiref==3.12.1\ndjango==6.1.2\n")
	fake.Responses[closureCall(root, "ruff")] = []byte("ruff==0.17.1\n")

	failing := func(_ context.Context, project, _ string) ([]string, error) {
		if project == "asgiref" {
			return nil, errors.New("PyPI unreachable")
		}
		return []string{"sha256:" + project}, nil
	}
	e := &Env{Cfg: config.Project{ProjectName: "demo"}, Runner: fake, Reporter: NopReporter{}, hashLookup: failing}
	e.deps.addRuntime("django==6.1.2")
	e.deps.addDev("ruff")

	err := e.writeRequirementsFiles(root)
	if err == nil || !strings.Contains(err.Error(), "PyPI unreachable") {
		t.Fatalf("writeRequirementsFiles = %v, want the failed lookup reported", err)
	}
	for _, name := range []string{RequirementsFile, DevRequirementsFile} {
		if _, statErr := os.Stat(filepath.Join(root, name)); !os.IsNotExist(statErr) {
			t.Errorf("%s was written without the hashes the run promised", name)
		}
	}
}
