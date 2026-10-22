package tui

import (
	"context"
	"errors"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"

	"github.com/vaibdix/django-cli/internal/config"
	"github.com/vaibdix/django-cli/internal/scaffold/pyenv"
)

type pythonOptionsMsg struct {
	options []pyenv.Option
	err     error
}

func (m *Model) discoverPythonCmd() tea.Cmd {
	run := m.runner()
	return func() tea.Msg {
		options, err := pyenv.Load(context.Background(), run)
		return pythonOptionsMsg{options: options, err: err}
	}
}

func (m *Model) applyPythonOptions(options []pyenv.Option, discoveryErr error) {
	m.PythonDiscoveryNote = ""
	if discoveryErr != nil {
		m.PythonDiscoveryNote = "Some interpreters may be missing: " + discoveryErr.Error()
	}
	m.AvailablePythonOptions = options
	if len(options) == 0 {
		m.SelectedPythonPath = ""
		m.SelectedPythonVersion = ""
		m.PythonForm = buildUnavailablePythonForm(&m.SelectedPythonPath, m.PythonDiscoveryNote)
		return
	}

	if m.SelectedPythonPath == "" || !m.hasPythonOption(m.SelectedPythonPath) {
		m.SelectedPythonPath = options[0].Path
	}
	m.syncSelectedPythonOption()
	m.PythonForm = m.buildPythonForm()
}

func buildPendingPythonForm(target *string) *huh.Form {
	return newForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Python Interpreter").
				Description("Looking for Python interpreters on this machine...").
				Options(huh.NewOption("Detecting...", "")).
				Validate(requirePython).
				Value(target),
		),
	)
}

func (m *Model) buildPythonForm() *huh.Form {
	options := make([]huh.Option[string], 0, len(m.AvailablePythonOptions))
	for _, option := range m.AvailablePythonOptions {
		options = append(options, huh.NewOption(option.Label, option.Path))
	}

	return newForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Python Interpreter").
				Description(withNote("Choose the Python runtime used for the virtualenv and Django project setup.", m.PythonDiscoveryNote)).
				Options(options...).
				Validate(requirePython).
				Value(&m.SelectedPythonPath),
		),
	)
}

func buildUnavailablePythonForm(target *string, note string) *huh.Form {
	if target != nil {
		*target = ""
	}
	return newForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Python Interpreter").
				Description(withNote("No supported Python 3.10+ interpreter was detected on this machine.", note)).
				Options(huh.NewOption("No Python 3.10+ interpreter found", "")).
				Validate(requirePython).
				Value(target),
		),
	)
}

func (m *Model) hasPythonOption(path string) bool {
	for _, option := range m.AvailablePythonOptions {
		if option.Path == path {
			return true
		}
	}
	return false
}

func (m *Model) syncSelectedPythonOption() {
	for _, option := range m.AvailablePythonOptions {
		if option.Path == m.SelectedPythonPath {
			m.SelectedPythonVersion = option.Version
			return
		}
	}
	if len(m.AvailablePythonOptions) > 0 {
		m.SelectedPythonPath = m.AvailablePythonOptions[0].Path
		m.SelectedPythonVersion = m.AvailablePythonOptions[0].Version
		return
	}
	m.SelectedPythonVersion = ""
}

func (m *Model) selectedPythonOption() (pyenv.Option, bool) {
	for _, option := range m.AvailablePythonOptions {
		if option.Path == m.SelectedPythonPath {
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
