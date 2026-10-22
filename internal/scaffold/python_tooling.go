package scaffold

import (
	"fmt"

	"github.com/vaibdix/django-stack/internal/config"
	"github.com/vaibdix/django-stack/internal/scaffold/pyenv"
)

type PythonToolingMetadata struct {
	RequiresPython    string
	RuffTargetVersion string
	MypyPythonVersion string
}

const fallbackPythonMinor = 10

func (e *Env) pythonToolingMetadata() PythonToolingMetadata {
	major, minor := 3, fallbackPythonMinor

	if selectedMajor, selectedMinor, _, _, ok := pyenv.ParseDetailedVersionOutput(e.Cfg.SelectedPythonVersion); ok {
		major = selectedMajor
		minor = selectedMinor
	}

	if floor, ok := config.PythonFloorForDjango(e.Cfg.DjangoVersion); ok && major == 3 && floor > minor {
		minor = floor
	}

	version := fmt.Sprintf("%d.%d", major, minor)
	return PythonToolingMetadata{
		RequiresPython:    ">=" + version,
		RuffTargetVersion: fmt.Sprintf("py%d%d", major, minor),
		MypyPythonVersion: version,
	}
}
