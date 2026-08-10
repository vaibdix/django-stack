package scaffold

import (
	"fmt"

	"github.com/vaibdix/django-cli/internal/scaffold/pyenv"
)

// PythonToolingMetadata holds the Python version strings that ruff, mypy and
// pyproject.toml each need in their own format.
type PythonToolingMetadata struct {
	RequiresPython    string
	RuffTargetVersion string
	MypyPythonVersion string
}

// pythonToolingMetadata derives the Python version strings that ruff, mypy and
// pyproject.toml need.
//
// It reads the version recorded in the config rather than re-inspecting the
// interpreter list: the list is a TUI concern, and by the time scaffolding runs
// the user's choice is already captured in SelectedPythonVersion.
func (e *Env) pythonToolingMetadata() PythonToolingMetadata {
	// A conservative floor for when no interpreter version was recorded.
	major, minor := 3, 10

	if selectedMajor, selectedMinor, _, _, ok := pyenv.ParseDetailedVersionOutput(e.Cfg.SelectedPythonVersion); ok {
		major = selectedMajor
		minor = selectedMinor
	}

	version := fmt.Sprintf("%d.%d", major, minor)
	return PythonToolingMetadata{
		RequiresPython:    ">=" + version,
		RuffTargetVersion: fmt.Sprintf("py%d%d", major, minor),
		MypyPythonVersion: version,
	}
}
