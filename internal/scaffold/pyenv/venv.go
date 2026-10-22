package pyenv

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/vaibdix/django-stack/internal/scaffold/command"
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

func CommandWorks(ctx context.Context, run command.Runner, name string) bool {
	if !CommandAvailable(name) {
		return false
	}
	output, err := probe(ctx, run, name, "--version")
	return err == nil && len(output) > 0
}

type Toolchain int

const (
	Pip Toolchain = iota
	UV
)

func (t Toolchain) String() string {
	if t == UV {
		return "uv"
	}
	return "pip"
}

func DetectToolchain(ctx context.Context, run command.Runner) Toolchain {
	ctx, cancel := context.WithTimeout(ctx, probeTimeout)
	defer cancel()

	output, err := run.Run(ctx, "", "uv", "--version")
	if err != nil || len(bytes.TrimSpace(output)) == 0 {
		return Pip
	}
	return UV
}

func PackageInstallCommand(t Toolchain, projectPath string) (string, []string) {
	pythonPath := InterpreterPath(projectPath)
	if t == UV {
		return "uv", []string{"pip", "install", "--python", pythonPath}
	}
	return pythonPath, []string{"-m", "pip", "install"}
}

func FreezeCommand(t Toolchain, projectPath string) (string, []string) {
	pythonPath := InterpreterPath(projectPath)
	if t == UV {
		return "uv", []string{"pip", "freeze", "--python", pythonPath}
	}
	return pythonPath, []string{"-m", "pip", "freeze"}
}

func PackageInstallArgs(t Toolchain, projectPath string, packages ...string) (string, []string) {
	name, baseArgs := PackageInstallCommand(t, projectPath)

	args := append([]string{}, baseArgs...)
	args = append(args, packages...)
	return name, args
}

func CreateVenv(ctx context.Context, run command.Runner, rep Reporter, projectPath, selectedPython string, t Toolchain) (Toolchain, error) {
	if selectedPython == "" {
		return t, errors.New("no Python interpreter was selected")
	}
	if !Works(ctx, run, selectedPython) {
		return t, fmt.Errorf("selected Python interpreter is not usable: %s", selectedPython)
	}

	if t == UV {
		step(rep, "Creating virtual environment with uv (fast)...")
		_, err := run.Run(ctx, projectPath, "uv", "venv", "--python", selectedPython, VenvDir)
		if err == nil {
			step(rep, "Virtual environment created.")
			return UV, nil
		}
		if ctx.Err() != nil {
			return t, fmt.Errorf("failed to create virtual environment with uv: %w", err)
		}
		step(rep, fmt.Sprintf("uv could not create the virtual environment (%v); "+
			"creating it with python -m venv and using pip for this project instead.", err))
		if err := os.RemoveAll(filepath.Join(projectPath, VenvDir)); err != nil {
			return t, fmt.Errorf("failed to remove the virtual environment uv left behind: %w", err)
		}
	}

	step(rep, "Creating virtual environment...")
	if _, err := run.Run(ctx, projectPath, selectedPython, "-m", "venv", VenvDir); err != nil {
		return Pip, fmt.Errorf("failed to create virtual environment: %w", err)
	}
	step(rep, "Virtual environment created.")
	return Pip, nil
}

func InstallAll(ctx context.Context, run command.Runner, rep Reporter, t Toolchain, projectPath string, packages ...string) error {
	progress(rep, "Installing Django and every package the project needs...")
	name, args := PackageInstallArgs(t, projectPath, packages...)
	if _, err := run.Run(ctx, projectPath, name, args...); err != nil {
		return fmt.Errorf("failed to install the project's packages: %w", err)
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

func ProbeVersion(ctx context.Context, run command.Runner, name string) (string, error) {
	output, err := probe(ctx, run, name, "--version")
	return string(output), err
}
