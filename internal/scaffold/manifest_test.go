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

func TestManifestRecordsWhatWasInstalledWithTheVersionsItResolved(t *testing.T) {
	t.Parallel()

	projectPath := t.TempDir()

	fake := commandtest.NewFake()
	name, args := pyenv.FreezeCommand(pyenv.Pip, projectPath)
	fake.Responses[commandtest.Call{Name: name, Args: args}.String()] = []byte(
		"asgiref==3.9.1\nDjango==6.1\ndjango-environ==0.14.0\ndjangorestframework==3.18.0\nmypy==2.4.0\nruff==0.16.2\n")

	e := &Env{
		Cfg:      mustSpec(config.Choices{Name: "demo", Django: "6.1", AddOns: config.NewSet(config.Ruff)}),
		Resolved: config.Resolved{PythonVersion: "Python 3.13.2"},
		ports:    port.Ports{Run: fake},
	}
	e.packages.AddRuntime("django==6.1", render.PkgEnviron, render.PkgRestFramework)
	e.packages.AddDev(render.PkgMypy)

	writeReport(t, projectPath, "runtime")
	writeReport(t, projectPath, "dev")

	e.dir = projectPath
	if err := e.writeDependencyManifest(t.Context()); err != nil {
		t.Fatalf("writing the manifest: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(projectPath, render.PyprojectFile))
	if err != nil {
		t.Fatalf("reading %s: %v", render.PyprojectFile, err)
	}
	manifest := string(content)

	for _, want := range []string{
		`"django==6.1"`,
		`"django-environ>=0.14.0,<0.15"`,
		`"djangorestframework>=3.18.0,<4"`,
		"[dependency-groups]",
		`"ruff>=0.16.2,<0.17"`,
		`"mypy>=2.4.0,<3"`,
	} {
		if !strings.Contains(manifest, want) {
			t.Errorf("manifest is missing %s:\n%s", want, manifest)
		}
	}

	if strings.Contains(manifest, "asgiref") {
		t.Errorf("a transitive dependency was declared as a direct one; that is the lockfile's job:\n%s", manifest)
	}

	version, err := os.ReadFile(filepath.Join(projectPath, render.PythonVersionFile))
	if err != nil {
		t.Fatalf("reading %s: %v", render.PythonVersionFile, err)
	}
	if strings.TrimSpace(string(version)) != "3.13.2" {
		t.Errorf("%s = %q, want the interpreter that was actually selected", render.PythonVersionFile, version)
	}
}

func TestAFailedFreezeFailsTheRunInsteadOfLeavingDependenciesUnpinned(t *testing.T) {
	t.Parallel()

	projectPath := t.TempDir()

	fake := commandtest.NewFake()
	name, args := pyenv.FreezeCommand(pyenv.Pip, projectPath)
	fake.FailWith(commandtest.Call{Name: name, Args: args}.String(), "no such file or directory")

	e := &Env{
		Cfg:      mustSpec(config.Choices{Name: "demo", Django: "6.1"}),
		Resolved: config.Resolved{PythonVersion: "Python 3.13.2"},
		ports:    port.Ports{Run: fake},
	}
	e.packages.AddRuntime("django==6.1", render.PkgEnviron)

	e.dir = projectPath
	err := e.writeDependencyManifest(t.Context())
	if err == nil || !strings.Contains(err.Error(), "installed package versions") {
		t.Fatalf("writeDependencyManifest = %v, want the failed freeze reported", err)
	}
	if _, statErr := os.Stat(filepath.Join(projectPath, render.PyprojectFile)); !os.IsNotExist(statErr) {
		t.Errorf("%s was written with unpinned dependencies after the freeze failed", render.PyprojectFile)
	}
}

func TestAFailedUVLockFailsTheRun(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	fake := commandtest.NewFake()
	fake.FailWith("uv lock", "No solution found when resolving dependencies")
	e := &Env{Cfg: mustSpec(config.Choices{Name: "demo"}), Toolchain: pyenv.UV, ports: port.Ports{Run: fake}}

	err := e.lockDependencies(t.Context(), root)
	if err == nil || !strings.Contains(err.Error(), render.LockFile) {
		t.Fatalf("lockDependencies = %v, want the missing lock file reported as a failure", err)
	}
}
