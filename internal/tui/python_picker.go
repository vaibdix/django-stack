package tui

import (
	"context"
	"errors"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"

	"github.com/vaibdix/django-stack/internal/config"
	"github.com/vaibdix/django-stack/internal/pyenv"
)

type pythonOptionsMsg struct {
	options []pyenv.Option
	err     error
}

func (m *Model) discoverPythonCmd() tea.Cmd {
	run := m.commandRunner()
	return func() tea.Msg {
		options, err := pyenv.Load(context.Background(), run)
		return pythonOptionsMsg{options: options, err: err}
	}
}

func (m *Model) applyPythonOptions(options []pyenv.Option, discoveryErr error) {
	m.pythonDiscoveryNote = ""
	if discoveryErr != nil {
		m.pythonDiscoveryNote = "Some interpreters may be missing: " + discoveryErr.Error()
	}
	m.availablePythonOptions = options
	if len(options) == 0 {
		m.project.SelectedPythonPath = ""
		m.project.SelectedPythonVersion = ""
		m.pythonForm = buildUnavailablePythonForm(&m.project.SelectedPythonPath, m.pythonDiscoveryNote)
		return
	}

	if m.project.SelectedPythonPath == "" || !m.hasPythonOption(m.project.SelectedPythonPath) {
		m.project.SelectedPythonPath = options[0].Path
	}
	m.syncSelectedPythonOption()
	m.pythonForm = m.buildPythonForm()
}

func buildPendingPythonForm(target *string) *huh.Form {
	return selectForm("Python Interpreter", "Looking for Python interpreters on this machine...",
		[]huh.Option[string]{huh.NewOption("Detecting...", "")}, target, requirePython)
}

func (m *Model) buildPythonForm() *huh.Form {
	options := make([]huh.Option[string], 0, len(m.availablePythonOptions))
	for _, option := range m.availablePythonOptions {
		options = append(options, huh.NewOption(option.Label, option.Path))
	}

	return selectForm("Python Interpreter",
		withNote("Choose the Python runtime used for the virtualenv and Django project setup.", m.pythonDiscoveryNote),
		options, &m.project.SelectedPythonPath, requirePython)
}

func buildUnavailablePythonForm(target *string, note string) *huh.Form {
	if target != nil {
		*target = ""
	}
	return selectForm("Python Interpreter", withNote("No supported Python 3.10+ interpreter was detected on this machine.", note),
		[]huh.Option[string]{huh.NewOption("No Python 3.10+ interpreter found", "")}, target, requirePython)
}

func (m *Model) hasPythonOption(path string) bool {
	for _, option := range m.availablePythonOptions {
		if option.Path == path {
			return true
		}
	}
	return false
}

func (m *Model) syncSelectedPythonOption() {
	for _, option := range m.availablePythonOptions {
		if option.Path == m.project.SelectedPythonPath {
			m.project.SelectedPythonVersion = option.Version
			return
		}
	}
	if len(m.availablePythonOptions) > 0 {
		m.project.SelectedPythonPath = m.availablePythonOptions[0].Path
		m.project.SelectedPythonVersion = m.availablePythonOptions[0].Version
		return
	}
	m.project.SelectedPythonVersion = ""
}

func (m *Model) selectedPythonOption() (pyenv.Option, bool) {
	for _, option := range m.availablePythonOptions {
		if option.Path == m.project.SelectedPythonPath {
			return option, true
		}
	}
	return pyenv.Option{}, false
}

func (m *Model) pythonNames() config.PythonNames {
	option, _ := m.selectedPythonOption()
	return config.PythonNames{Stdlib: option.Stdlib, Keywords: option.Keywords}
}

func requirePython(path string) error {
	if path == "" {
		return errors.New("no usable Python interpreter yet; install one (for example `uv python install 3.13`) and start django-stack again")
	}
	return nil
}

func withNote(description, note string) string {
	if note == "" {
		return description
	}
	return description + " " + note
}
