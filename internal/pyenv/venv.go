package pyenv

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/vaibdix/django-stack/internal/command"
	"github.com/vaibdix/django-stack/internal/report"
)

const VenvDir = ".venv"

func InterpreterPath(projectPath string) string {
	return filepath.Join(projectPath, VenvDir, VenvBin(runtime.GOOS), "python"+ExeSuffix(runtime.GOOS))
}

func (h Host) CommandAvailable(name string) bool {
	_, err := h.LookPath(name)
	return err == nil
}

func (h Host) CommandWorks(ctx context.Context, run command.Runner, name string) bool {
	if !h.CommandAvailable(name) {
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

func CreateVenv(ctx context.Context, run command.Runner, rep report.Steps, projectPath, selectedPython string, t Toolchain) (Toolchain, error) {
	if selectedPython == "" {
		return t, errors.New("no Python interpreter was selected")
	}
	if !Works(ctx, run, selectedPython) {
		return t, fmt.Errorf("selected Python interpreter is not usable: %s", selectedPython)
	}

	if t == UV {
		rep.Step("Creating virtual environment with uv (fast)...")
		_, err := run.Run(ctx, projectPath, "uv", "venv", "--python", selectedPython, VenvDir)
		if err == nil {
			rep.Step("Virtual environment created.")
			return UV, nil
		}
		if ctx.Err() != nil {
			return t, fmt.Errorf("create virtual environment with uv: %w", err)
		}
		rep.Step(fmt.Sprintf("uv could not create the virtual environment (%v); "+
			"creating it with python -m venv and using pip for this project instead.", err))
		if err := os.RemoveAll(filepath.Join(projectPath, VenvDir)); err != nil {
			return t, fmt.Errorf("remove the virtual environment uv left behind: %w", err)
		}
	}

	rep.Step("Creating virtual environment...")
	if _, err := run.Run(ctx, projectPath, selectedPython, "-m", "venv", VenvDir); err != nil {
		return Pip, fmt.Errorf("create virtual environment: %w", err)
	}
	rep.Step("Virtual environment created.")
	return Pip, nil
}

func InstallAll(ctx context.Context, run command.Runner, rep report.Steps, t Toolchain, projectPath string, packages ...string) error {
	rep.Step("Installing Django and every package the project needs...")
	name, args := PackageInstallArgs(t, projectPath, packages...)
	if _, err := run.Run(ctx, projectPath, name, args...); err != nil {
		return fmt.Errorf("install the project's packages: %w", err)
	}
	return nil
}

func ProbeVersion(ctx context.Context, run command.Runner, name string) (string, error) {
	output, err := probe(ctx, run, name, "--version")
	return string(output), err
}
