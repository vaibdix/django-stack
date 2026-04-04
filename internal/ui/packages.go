package ui

import (
	"fmt"
	"os"
	"runtime"
)

// installAndReport installs pip packages and reports the result via step messages.
// It handles the Windows/Unix branching for command output.
func (m *Model) installAndReport(projectPath string, label string, packages ...string) error {
	cmd := installPackages(projectPath, packages...)

	if runtime.GOOS == "windows" {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to install %s: %w", label, err)
		}
	} else {
		output, err := cmd.CombinedOutput()
		if err != nil {
			m.AppendStepMessage(fmt.Sprintf("Warning: Failed to install %s: %v\nOutput: %s", label, err, string(output)))
			return fmt.Errorf("failed to install %s: %w", label, err)
		}
	}

	m.AppendStepMessage(fmt.Sprintf("%s installed.", label))
	if isUvAvailable() {
		m.AppendStepMessage(fmt.Sprintf("%s was installed into %s using uv pip.", label, VenvDir))
	}
	return nil
}
