package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"

	"github.com/vaibdix/django-cli/internal/scaffold/pyenv"
)

// pythonOptionsMsg carries the result of interpreter discovery back to Update.
type pythonOptionsMsg []pyenv.Option

// discoverPythonCmd runs interpreter discovery off the update goroutine.
//
// Discovery execs every candidate interpreter on the machine to read its
// version. Doing that inside NewModel made construction impure — the model
// could not be built in a hermetic test — and stalled the splash screen behind
// a dozen subprocesses. As a tea.Cmd it runs while the splash counts down.
func discoverPythonCmd() tea.Cmd {
	return func() tea.Msg { return pythonOptionsMsg(pyenv.Load()) }
}

// applyPythonOptions folds discovered interpreters into the model and rebuilds
// the picker. It performs no I/O, so it is safe on the update goroutine.
func (m *Model) applyPythonOptions(options []pyenv.Option) {
	m.AvailablePythonOptions = options
	if len(options) == 0 {
		m.SelectedPythonPath = ""
		m.SelectedPythonVersion = ""
		m.PythonForm = buildUnavailablePythonForm(&m.SelectedPythonPath)
		return
	}

	if m.SelectedPythonPath == "" || !m.hasPythonOption(m.SelectedPythonPath) {
		m.SelectedPythonPath = options[0].Path
	}
	m.syncSelectedPythonOption()
	m.PythonForm = m.buildPythonForm()
}

// buildPendingPythonForm is what the picker shows before discovery finishes.
// It is replaced by applyPythonOptions as soon as the results arrive.
func buildPendingPythonForm(target *string) *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Python Interpreter").
				Description("Looking for Python interpreters on this machine...").
				Options(huh.NewOption("Detecting...", "")).
				Value(target),
		),
	)
}

func (m *Model) buildPythonForm() *huh.Form {
	options := make([]huh.Option[string], 0, len(m.AvailablePythonOptions))
	for _, option := range m.AvailablePythonOptions {
		options = append(options, huh.NewOption(option.Label, option.Path))
	}

	return huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Python Interpreter").
				Description("Choose the Python runtime used for the virtualenv and Django project setup.").
				Options(options...).
				Value(&m.SelectedPythonPath),
		),
	)
}

func buildUnavailablePythonForm(target *string) *huh.Form {
	if target != nil {
		*target = ""
	}
	return huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Python Interpreter").
				Description("No supported Python 3.8+ interpreter was detected on this machine.").
				Options(huh.NewOption("No Python 3.8+ interpreter found", "")).
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
