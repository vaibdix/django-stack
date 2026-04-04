package ui

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

func (m *Model) CreateVirtualEnvironment(projectPath string) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultCommandTimeout)
	defer cancel()

	if m.SelectedPythonPath != "" {
		if !verifyPythonWorks(m.SelectedPythonPath) {
			return fmt.Errorf("selected Python interpreter is not usable: %s", m.SelectedPythonPath)
		}
		if isUvAvailable() {
			m.AppendStepMessage("Creating virtual environment with uv (fast)...")
			cmd := exec.CommandContext(ctx, "uv", "venv", "--python", m.SelectedPythonPath, VenvDir)
			cmd.Dir = projectPath
			if output, err := cmd.CombinedOutput(); err != nil {
				return fmt.Errorf("failed to create virtual environment with uv: %w\nOutput: %s", err, string(output))
			}
		} else {
			m.AppendStepMessage("Creating virtual environment...")
			cmd := exec.CommandContext(ctx, m.SelectedPythonPath, "-m", "venv", VenvDir)
			cmd.Dir = projectPath
			if output, err := cmd.CombinedOutput(); err != nil {
				return fmt.Errorf("failed to create virtual environment: %w\nOutput: %s", err, string(output))
			}
		}
		m.AppendStepMessage("Virtual environment created.")
		return nil
	}

	pyenvIssue := hasPyenvVersionError()
	pyenvFixed := false

	if pyenvIssue {
		m.AppendStepMessage("Detected pyenv configuration issue...")
		if attemptFixPyenv() {
			m.AppendStepMessage("Auto-fixed pyenv configuration!")
			pyenvFixed = true
		}
	}

	pythonPath, found := findWorkingPython(pyenvIssue && !pyenvFixed)

	if pyenvIssue && !pyenvFixed {
		if found {
			m.AppendStepMessage("Using direct Python path to bypass pyenv...")
			cmd := exec.CommandContext(ctx, pythonPath, "-m", "venv", VenvDir)
			cmd.Dir = projectPath
			if output, err := cmd.CombinedOutput(); err != nil {
				return fmt.Errorf("failed to create virtual environment: %w\nOutput: %s", err, string(output))
			}
			m.AppendStepMessage("Virtual environment created.")
			warnUserAboutPyenv()
			return nil
		}
		warnUserAboutPyenv()
		return fmt.Errorf("pyenv is misconfigured and no working Python found.\n\n" +
			"Run one of these commands to fix:\n" +
			"  pyenv global system\n" +
			"  pyenv install 3.11.0 && pyenv global 3.11.0")
	}

	if isUvAvailable() && found {
		m.AppendStepMessage("Creating virtual environment with uv (fast)...")
		cmd := exec.CommandContext(ctx, "uv", "venv", "--python", pythonPath, VenvDir)
		cmd.Dir = projectPath
		cmd.Env = append(os.Environ(), "VIRTUAL_ENV="+projectPath)
		if output, err := cmd.CombinedOutput(); err != nil {
			if strings.Contains(string(output), "No global") ||
				strings.Contains(string(output), "no global") {
				m.AppendStepMessage("uv failed due to pyenv issue. Trying direct Python...")
				cmd = exec.CommandContext(ctx, pythonPath, "-m", "venv", VenvDir)
				cmd.Dir = projectPath
				if output2, err2 := cmd.CombinedOutput(); err2 != nil {
					return fmt.Errorf("failed to create virtual environment: %w\nOutput: %s", err2, string(output2))
				}
				m.AppendStepMessage("Virtual environment created.")
				warnUserAboutPyenv()
				return nil
			}
			return fmt.Errorf("failed to create virtual environment with uv: %w\nOutput: %s", err, string(output))
		}
		m.AppendStepMessage("Virtual environment created.")
		return nil
	}

	if !found {
		pythonPath, found = findWorkingPython(false)
	}

	if !found {
		if runtime.GOOS == "windows" {
			return fmt.Errorf("Python not found. Please:\n" +
				"1. Download Python from https://www.python.org/downloads/\n" +
				"2. During installation, CHECK 'Add Python to PATH'\n" +
				"3. Restart your terminal/command prompt\n" +
				"4. Run this command again\n\n" +
				"Or install uv for automatic Python detection: pip install uv")
		}
		return fmt.Errorf("Python 3.8+ not found. Please install Python and ensure it's in your PATH")
	}

	m.AppendStepMessage("Creating virtual environment...")
	cmd := exec.CommandContext(ctx, pythonPath, "-m", "venv", VenvDir)
	cmd.Dir = projectPath
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create virtual environment: %w\nOutput: %s", err, string(output))
	}
	m.AppendStepMessage("Virtual environment created.")
	return nil
}

func (m *Model) InstallDjango(projectPath string) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultCommandTimeout)
	defer cancel()

	packages := []string{
		"django==" + m.DjangoVersion,
		PkgBrowserReload,
		PkgEnviron,
		PkgDotenv,
		PkgPsycopg,
		PkgArgon2,
	}

	m.UpdateProgress("Installing Django...")
	cmd := installPackages(projectPath, packages...)
	cmd = exec.CommandContext(ctx, cmd.Args[0], cmd.Args[1:]...)
	cmd.Dir = projectPath

	if runtime.GOOS == "windows" {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to install Django packages: %w", err)
		}
	} else {
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to install Django packages: %w\nOutput: %s", err, string(output))
		}
	}

	m.AppendStepMessage("Django installed.")
	m.AppendStepMessage("django-environ installed.")
	m.AppendStepMessage("python-dotenv installed.")
	m.AppendStepMessage("psycopg2-binary installed.")
	m.AppendStepMessage("argon2-cffi installed.")
	if isUvAvailable() {
		m.AppendStepMessage("Packages were installed into " + VenvDir + " using uv pip.")
	} else {
		m.AppendStepMessage("Packages were installed into " + VenvDir + " using pip.")
	}
	m.UpdateProgress("Installing development dependencies...")

	return nil
}
