package pyenv

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/vaibdix/django-cli/internal/scaffold/command"
)

const VenvDir = ".venv"

type Reporter interface {
	Step(message string)
	Progress(status string)
}

func InterpreterPath(projectPath string) string {
	if runtime.GOOS == "windows" {
		return filepath.Join(projectPath, VenvDir, "Scripts", "python.exe")
	}
	return filepath.Join(projectPath, VenvDir, "bin", "python")
}

func CommandAvailable(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func CommandWorks(name string) bool {
	if !CommandAvailable(name) {
		return false
	}
	output, err := probe(name, "--version")
	return err == nil && len(output) > 0
}

func UVAvailable() bool {
	if !CommandAvailable("uv") {
		return false
	}
	output, err := probe("uv", "--version")
	return err == nil && len(output) > 0
}

func PackageInstallCommand(projectPath string) (string, []string) {
	pythonPath := InterpreterPath(projectPath)
	if UVAvailable() {
		return "uv", []string{"pip", "install", "--python", pythonPath}
	}
	return pythonPath, []string{"-m", "pip", "install"}
}

func FreezeCommand(projectPath string) (string, []string) {
	pythonPath := InterpreterPath(projectPath)
	if UVAvailable() {
		return "uv", []string{"pip", "freeze", "--python", pythonPath}
	}
	return pythonPath, []string{"-m", "pip", "freeze"}
}

func PackageInstallArgs(projectPath string, packages ...string) (string, []string) {
	name, baseArgs := PackageInstallCommand(projectPath)

	args := append([]string{}, baseArgs...)
	args = append(args, packages...)
	if name != "uv" && runtime.GOOS == "windows" {
		args = append(args, "--progress-bar", "pretty")
	}
	return name, args
}

func CreateVenv(ctx context.Context, run command.Runner, rep Reporter, projectPath, selectedPython string) error {
	if selectedPython == "" {
		return errors.New("no Python interpreter was selected")
	}
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

func ProbeVersion(name string) (string, error) {
	output, err := probe(name, "--version")
	return string(output), err
}
