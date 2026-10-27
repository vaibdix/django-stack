package scaffold

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vaibdix/django-stack/internal/command/commandtest"
	"github.com/vaibdix/django-stack/internal/config"
	"github.com/vaibdix/django-stack/internal/port"
	"github.com/vaibdix/django-stack/internal/pyenv"
	"github.com/vaibdix/django-stack/internal/render"
)

func TestEveryGeneratedJSONFileEndsWithANewline(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	cfg := config.Choices{Name: "demo", App: "catalog", Stack: config.StackTailwind}
	if err := writeVSCodeConfig(root, cfg, render.NPM); err != nil {
		t.Fatalf("writeVSCodeConfig: %v", err)
	}
	e := &Env{Cfg: mustSpec(cfg)}
	if err := e.at(root).ensurePackageJSON(); err != nil {
		t.Fatalf("ensurePackageJSON: %v", err)
	}
	if err := e.at(root).updatePackageJSONForTailwind(); err != nil {
		t.Fatalf("updatePackageJSONForTailwind: %v", err)
	}

	for _, name := range []string{".vscode/tasks.json", ".vscode/launch.json", ".vscode/extensions.json", "package.json"} {
		data, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			t.Fatalf("reading %s: %v", name, err)
		}
		if !strings.HasSuffix(string(data), "}\n") {
			t.Errorf("%s does not end with a newline, so end-of-file-fixer rewrites it on the first commit", name)
		}
	}
}

func TestPreCommitSetupNeverAutoupdatesHooks(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	fake := commandtest.NewFake()
	e := &Env{Cfg: mustSpec(config.Choices{Name: "demo", AddOns: config.NewSet(config.PreCommit)}), ports: port.Ports{Run: fake}}
	e.dir = root
	if err := e.applyPreCommit(t.Context()); err != nil {
		t.Fatalf("applyPreCommit: %v", err)
	}
	for _, line := range fake.CommandLines() {
		if strings.Contains(line, "autoupdate") {
			t.Errorf("generation rewrites hook versions from the network: %s", line)
		}
	}
}

func TestAFailedPreCommitInstallFailsTheStep(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	fake := commandtest.NewFake()
	fake.FailWith(pyenv.InterpreterPath(root)+" -m pre_commit install", "not a git repository")
	e := &Env{Cfg: mustSpec(config.Choices{Name: "demo", AddOns: config.NewSet(config.Git, config.PreCommit)}), ports: port.Ports{Run: fake}}

	e.dir = root
	err := e.installGitHooks(t.Context())
	if err == nil || !strings.Contains(err.Error(), "pre-commit hooks") {
		t.Fatalf("installGitHooks = %v, want the failed install reported", err)
	}
}
