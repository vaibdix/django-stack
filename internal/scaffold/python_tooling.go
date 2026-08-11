package scaffold

import (
	"fmt"

	"github.com/vaibdix/django-cli/internal/scaffold/pyenv"
)

type PythonToolingMetadata struct {
	RequiresPython    string
	RuffTargetVersion string
	MypyPythonVersion string
}

func (e *Env) pythonToolingMetadata() PythonToolingMetadata {

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
