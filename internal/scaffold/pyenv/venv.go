package pyenv

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/vaibdix/django-cli/internal/scaffold/command"
)

// VenvDir is the directory a project's virtual environment lives in.
const VenvDir = ".venv"

// Reporter receives human-readable progress. Declared here rather than imported
// so pyenv stays a leaf; scaffold.Env satisfies it structurally.
type Reporter interface {
	Step(message string)
	Progress(status string)
}

// InterpreterPath returns the path to the Python interpreter inside a
// project's virtualenv, using the platform's layout.
func InterpreterPath(projectPath string) string {
	if runtime.GOOS == "windows" {
		return filepath.Join(projectPath, VenvDir, "Scripts", "python.exe")
	}
	return filepath.Join(projectPath, VenvDir, "bin", "python")
}

// CommandAvailable reports whether name is on PATH.
func CommandAvailable(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// UVAvailable reports whether the uv installer is available. uv is much faster
// than pip, so it is preferred when present.
func UVAvailable() bool {
	if !CommandAvailable("uv") {
		return false
	}
	output, err := probe("uv", "--version")
	return err == nil && len(output) > 0
}

// PackageInstallCommand returns the base executable and arguments used to
// install packages into the project venv.
func PackageInstallCommand(projectPath string) (string, []string) {
	pythonPath := InterpreterPath(projectPath)
	if UVAvailable() {
		return "uv", []string{"pip", "install", "--python", pythonPath}
	}
	return pythonPath, []string{"-m", "pip", "install"}
}

// PackageInstallArgs returns the executable and full argument list for
// installing packages into the project's virtualenv, preferring uv when
// available.
// PackageInstallArgs returns the full command line for installing packages.
func PackageInstallArgs(projectPath string, packages ...string) (string, []string) {
	name, baseArgs := PackageInstallCommand(projectPath)

	args := append([]string{}, baseArgs...)
	args = append(args, packages...)
	if name != "uv" && runtime.GOOS == "windows" {
		args = append(args, "--progress-bar", "pretty")
	}
	return name, args
}

// CreateVenv creates the project's virtual environment, preferring uv when it
// is available and falling back to `python -m venv`.
func CreateVenv(ctx context.Context, run command.Runner, rep Reporter, projectPath, selectedPython string) error {
	if selectedPython != "" {
		if !Works(selectedPython) {
			return fmt.Errorf("selected Python interpreter is not usable: %s", selectedPython)
		}
		if UVAvailable() {
			step(rep, "Creating virtual environment with uv (fast)...")
			if _, err := run.Run(ctx, projectPath, "uv", "venv", "--python", selectedPython, VenvDir); err != nil {
				return fmt.Errorf("failed to create virtual environment with uv: %w", err)
			}
		} else {
			step(rep, "Creating virtual environment...")
			if _, err := run.Run(ctx, projectPath, selectedPython, "-m", "venv", VenvDir); err != nil {
				return fmt.Errorf("failed to create virtual environment: %w", err)
			}
		}
		step(rep, "Virtual environment created.")
		return nil
	}

	pyenvIssue := HasVersionError()
	pyenvFixed := false

	if pyenvIssue {
		step(rep, "Detected pyenv configuration issue...")
		if AttemptFix() {
			step(rep, "Auto-fixed pyenv configuration!")
			pyenvFixed = true
		}
	}

	pythonPath, found := FindWorking(pyenvIssue && !pyenvFixed)

	if pyenvIssue && !pyenvFixed {
		if found {
			step(rep, "Using direct Python path to bypass pyenv...")
			if _, err := run.Run(ctx, projectPath, pythonPath, "-m", "venv", VenvDir); err != nil {
				return fmt.Errorf("failed to create virtual environment: %w", err)
			}
			step(rep, "Virtual environment created.")
			warnUserAboutPyenv()
			return nil
		}
		warnUserAboutPyenv()
		return fmt.Errorf("pyenv is misconfigured and no working Python found.\n\n" +
			"Run one of these commands to fix:\n" +
			"  pyenv global system\n" +
			"  pyenv install 3.11.0 && pyenv global 3.11.0")
	}

	if UVAvailable() && found {
		step(rep, "Creating virtual environment with uv (fast)...")
		output, err := run.Run(ctx, projectPath, "uv", "venv", "--python", pythonPath, VenvDir)
		if err != nil {
			// uv reports a pyenv "no global version" problem in its output; fall
			// back to the interpreter directly rather than failing the scaffold.
			if strings.Contains(string(output), "No global") ||
				strings.Contains(string(output), "no global") {
				step(rep, "uv failed due to a pyenv issue. Trying direct Python...")
				if _, err := run.Run(ctx, projectPath, pythonPath, "-m", "venv", VenvDir); err != nil {
					return fmt.Errorf("failed to create virtual environment: %w", err)
				}
				step(rep, "Virtual environment created.")
				warnUserAboutPyenv()
				return nil
			}
			return fmt.Errorf("failed to create virtual environment with uv: %w", err)
		}
		step(rep, "Virtual environment created.")
		return nil
	}

	if !found {
		pythonPath, found = FindWorking(false)
	}

	if !found {
		if runtime.GOOS == "windows" {
			return fmt.Errorf("python not found; to fix:\n" +
				"1. Download Python from https://www.python.org/downloads/\n" +
				"2. During installation, CHECK 'Add Python to PATH'\n" +
				"3. Restart your terminal/command prompt\n" +
				"4. Run this command again\n\n" +
				"Or install uv for automatic Python detection: pip install uv")
		}
		return fmt.Errorf("python 3.8+ not found; install it and ensure it is on your PATH")
	}

	step(rep, "Creating virtual environment...")
	if _, err := run.Run(ctx, projectPath, pythonPath, "-m", "venv", VenvDir); err != nil {
		return fmt.Errorf("failed to create virtual environment: %w", err)
	}
	step(rep, "Virtual environment created.")
	return nil
}

// InstallDjango installs the pinned Django release plus the baseline runtime
// dependencies into the project's virtual environment.
//
// The caller supplies the dependency list so this package does not need to know
// which extras a given scaffold wants.
func InstallDjango(ctx context.Context, run command.Runner, rep Reporter, projectPath, djangoVersion string, extras ...string) error {
	packages := append([]string{"django==" + djangoVersion}, extras...)

	progress(rep, "Installing Django...")
	name, args := PackageInstallArgs(projectPath, packages...)

	if _, err := run.Run(ctx, projectPath, name, args...); err != nil {
		return fmt.Errorf("failed to install Django packages: %w", err)
	}

	step(rep, "Django installed.")
	if UVAvailable() {
		step(rep, "Packages were installed into "+VenvDir+" using uv pip.")
	} else {
		step(rep, "Packages were installed into "+VenvDir+" using pip.")
	}

	return nil
}

// Install installs pip packages into the project venv and reports the outcome.
func Install(ctx context.Context, run command.Runner, rep Reporter, projectPath, label string, packages ...string) error {
	name, args := PackageInstallArgs(projectPath, packages...)

	if _, err := run.Run(ctx, projectPath, name, args...); err != nil {
		step(rep, fmt.Sprintf("Warning: failed to install %s: %v", label, err))
		return fmt.Errorf("failed to install %s: %w", label, err)
	}

	step(rep, label+" installed.")
	if UVAvailable() {
		step(rep, fmt.Sprintf("%s was installed into %s using uv pip.", label, VenvDir))
	}
	return nil
}

func step(rep Reporter, message string) {
	if rep != nil {
		rep.Step(message)
	}
}

func progress(rep Reporter, status string) {
	if rep != nil {
		rep.Progress(status)
	}
}
