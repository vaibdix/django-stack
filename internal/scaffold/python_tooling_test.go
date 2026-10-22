package scaffold

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/vaibdix/django-stack/internal/config"
	"github.com/vaibdix/django-stack/internal/pyenv"
)

func TestAnUnparseableInterpreterVersionIsNotGuessed(t *testing.T) {
	t.Parallel()

	e := &Env{cfg: config.Project{ProjectName: "demo", SelectedPythonVersion: "unknown"}}
	if err := e.writePythonVersionFile(t.TempDir()); err == nil {
		t.Error(".python-version was written from a guess instead of the selected interpreter")
	}
}

func TestPackageInstallCommandUsesVenvPythonWithoutUV(t *testing.T) {
	t.Parallel()

	projectPath := filepath.Join(t.TempDir(), "demo")
	command, args := pyenv.PackageInstallCommand(pyenv.Pip, projectPath)

	if command != pyenv.InterpreterPath(projectPath) {
		t.Fatalf("pip installs must run the venv's own interpreter, got %q", command)
	}
	if len(args) < 3 || args[0] != "-m" || args[1] != "pip" || args[2] != "install" {
		t.Fatalf("expected pip install args, got %#v", args)
	}
}

func TestInstallPackagesUsesVenvTargetWithUV(t *testing.T) {
	t.Parallel()

	projectPath := filepath.Join(t.TempDir(), "demo")
	path, args := pyenv.PackageInstallArgs(pyenv.UV, projectPath, "django==6.0.3")
	joined := strings.Join(args, " ")

	if path != "uv" {
		t.Fatalf("expected the uv command, got path=%q args=%#v", path, args)
	}
	if !strings.Contains(joined, "--python "+pyenv.InterpreterPath(projectPath)) {
		t.Fatalf("uv pip must target the venv's interpreter, got %q", joined)
	}
	if strings.Contains(joined, "--system") {
		t.Fatalf("install command must not target system Python: %q", joined)
	}
}

func TestPythonToolingMetadataMatchesSelectedInterpreter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		option          pyenv.Option
		wantRequires    string
		wantRuffTarget  string
		wantMypyVersion string
	}{
		{
			name:            "python_310",
			option:          pyenv.Option{Path: "/tmp/python310", Version: "3.10.18", Major: 3, Minor: 10, Patch: 18},
			wantRequires:    ">=3.10",
			wantRuffTarget:  "py310",
			wantMypyVersion: "3.10",
		},
		{
			name:            "python_312",
			option:          pyenv.Option{Path: "/tmp/python312", Version: "3.12.4", Major: 3, Minor: 12, Patch: 4},
			wantRequires:    ">=3.12",
			wantRuffTarget:  "py312",
			wantMypyVersion: "3.12",
		},
		{
			name:            "python_314",
			option:          pyenv.Option{Path: "/tmp/python314", Version: "3.14.3", Major: 3, Minor: 14, Patch: 3},
			wantRequires:    ">=3.14",
			wantRuffTarget:  "py314",
			wantMypyVersion: "3.14",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := &Env{
				cfg: config.Project{
					SelectedPythonPath:    tt.option.Path,
					SelectedPythonVersion: tt.option.Version,
				},
			}

			requires, ruffTarget, mypyVersion := pythonToolingMetadataOf(env)
			if requires != tt.wantRequires || ruffTarget != tt.wantRuffTarget || mypyVersion != tt.wantMypyVersion {
				t.Fatalf("unexpected tooling metadata for %s: got (%s, %s, %s), want (%s, %s, %s)", tt.name, requires, ruffTarget, mypyVersion, tt.wantRequires, tt.wantRuffTarget, tt.wantMypyVersion)
			}
		})
	}
}
