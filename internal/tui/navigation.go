package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
)

var previousStep = map[Step]Step{
	StepPythonSelection: StepProjectName,
	StepDjangoVersion:   StepPythonSelection,
	StepDatabase:        StepDjangoVersion,
	StepProjectConfig:   StepDatabase,
	StepAppName:         StepProjectConfig,
}

func isQuestionStep(step Step) bool {
	switch step {
	case StepProjectName, StepPythonSelection, StepDjangoVersion,
		StepDatabase, StepProjectConfig, StepAppName:
		return true
	default:
		return false
	}
}

func (m *Model) handleStepKeys(msg tea.Msg) ([]tea.Cmd, bool) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok || !isQuestionStep(m.Step) {
		return nil, false
	}

	switch keyMsg.Type {
	case tea.KeyLeft:
		return m.goToPreviousStep(), true
	case tea.KeyRight:
		return m.updateActiveForm(tea.KeyMsg{Type: tea.KeyEnter}), true
	default:
		return nil, false
	}
}

func (m *Model) goToPreviousStep() []tea.Cmd {
	target, ok := previousStep[m.Step]
	if !ok {
		return nil
	}

	form := m.rebuildForm(target)
	m.Step = target
	m.syncFormWidths()

	if form == nil {
		return nil
	}
	return []tea.Cmd{form.Init()}
}

func (m *Model) rebuildForm(step Step) *huh.Form {
	switch step {
	case StepProjectName:
		m.ProjectNameForm = buildProjectNameForm(&m.ProjectName)
		return m.ProjectNameForm

	case StepPythonSelection:
		if len(m.AvailablePythonOptions) == 0 {
			m.PythonForm = buildUnavailablePythonForm(&m.SelectedPythonPath)
		} else {
			m.PythonForm = m.buildPythonForm()
		}
		return m.PythonForm

	case StepDjangoVersion:
		if len(m.AvailableDjangoOptions) == 0 {
			m.DjangoVersionForm = buildUnavailableDjangoForm(&m.DjangoVersion, m.DjangoVersionNote)
		} else {
			m.DjangoVersionForm = m.buildDjangoVersionForm()
		}
		return m.DjangoVersionForm

	case StepDatabase:
		m.DatabaseForm = buildDatabaseForm(&m.DatabaseType)
		return m.DatabaseForm

	case StepProjectConfig:
		m.ProjectConfigForm = buildProjectConfigForm(&m.SelectedOptions)
		return m.ProjectConfigForm

	case StepAppName:
		m.AppNameForm = buildAppNameForm(&m.AppName)
		return m.AppNameForm
	}

	return nil
}
