package scaffold

import (
	"fmt"

	"github.com/vaibdix/django-stack/internal/config"
	"github.com/vaibdix/django-stack/internal/pyenv"
)

type pythonToolingMetadata struct {
	RequiresPython    string
	RuffTargetVersion string
	MypyPythonVersion string
}

const fallbackPythonMinor = 10

func (e *Env) pythonToolingMetadata() pythonToolingMetadata {
	major, minor := 3, fallbackPythonMinor

	if selectedMajor, selectedMinor, _, _, ok := pyenv.ParseDetailedVersionOutput(e.resolved.PythonVersion); ok {
		major = selectedMajor
		minor = selectedMinor
	}

	if floor, ok := config.PythonFloorForDjango(e.cfg.Django); ok && major == 3 && floor > minor {
		minor = floor
	}

	version := fmt.Sprintf("%d.%d", major, minor)
	return pythonToolingMetadata{
		RequiresPython:    ">=" + version,
		RuffTargetVersion: fmt.Sprintf("py%d%d", major, minor),
		MypyPythonVersion: version,
	}
}
