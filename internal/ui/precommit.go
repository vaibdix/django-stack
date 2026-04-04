package ui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// SetupPreCommitHooks installs pre-commit, generates .pre-commit-config.yaml,
// and runs `pre-commit install`.
func (m *Model) SetupPreCommitHooks(projectPath string) error {
	m.UpdateProgress("Setting up pre-commit hooks...")

	if err := m.installAndReport(projectPath, "pre-commit", PkgPreCommit); err != nil {
		return err
	}

	config := m.generatePreCommitConfig()

	configPath := filepath.Join(projectPath, PreCommitConfigFile)
	if err := os.WriteFile(configPath, []byte(config), 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", PreCommitConfigFile, err)
	}
	m.AppendStepMessage("Created: " + PreCommitConfigFile)

	// Run pre-commit install using the venv python
	pythonPath := getPythonPath(projectPath)
	cmd := exec.Command(pythonPath, "-m", "pre_commit", "install")
	cmd.Dir = projectPath
	if runtime.GOOS == "windows" {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			m.AppendStepMessage("Warning: pre-commit install failed: " + err.Error())
		}
	} else {
		if output, err := cmd.CombinedOutput(); err != nil {
			m.AppendStepMessage(fmt.Sprintf("Warning: pre-commit install failed: %v\nOutput: %s", err, string(output)))
		}
	}

	m.AppendStepMessage("Pre-commit hooks installed")
	return nil
}

func (m *Model) generatePreCommitConfig() string {
	config := `repos:
  - repo: https://github.com/pre-commit/pre-commit-hooks
    rev: v4.6.0
    hooks:
      - id: trailing-whitespace
      - id: end-of-file-fixer
      - id: check-yaml
      - id: debug-statements
`

	if m.SetupRuff {
		config += `
  - repo: https://github.com/astral-sh/ruff-pre-commit
    rev: v0.8.0
    hooks:
      - id: ruff
        args: [--fix]
      - id: ruff-format
`
	}

	// Extract major.minor from DjangoVersion (e.g., "6.0.3" → "6.0")
	targetVersion := "5.0"
	if m.DjangoVersion != "" {
		parts := strings.SplitN(m.DjangoVersion, ".", 3)
		if len(parts) >= 2 {
			targetVersion = parts[0] + "." + parts[1]
		}
	}

	config += fmt.Sprintf(`
  - repo: https://github.com/Riverside-Healthcare/djLint
    rev: v1.36.1
    hooks:
      - id: djlint-django

  - repo: https://github.com/adamchainz/django-upgrade
    rev: "1.21.0"
    hooks:
      - id: django-upgrade
        args: [--target-version, "%s"]
`, targetVersion)
	return config
}
