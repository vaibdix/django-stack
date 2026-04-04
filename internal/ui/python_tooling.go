package ui

import "fmt"

type PythonToolingMetadata struct {
	RequiresPython    string
	RuffTargetVersion string
	MypyPythonVersion string
}

func (m *Model) pythonToolingMetadata() PythonToolingMetadata {
	major, minor := 3, 10

	if option, ok := m.selectedPythonOption(); ok && option.Major > 0 && option.Minor > 0 {
		major = option.Major
		minor = option.Minor
	} else if selectedMajor, selectedMinor, _, _, ok := parseDetailedPythonVersionOutput(m.SelectedPythonVersion); ok {
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
