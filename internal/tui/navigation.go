package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
)

func (m *Model) handleStepKeys(msg tea.Msg) ([]tea.Cmd, bool) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok || !isQuestionStep(m.Step) {
		return nil, false
	}

	switch {
	case keyMsg.Type == tea.KeyShiftTab:
		return m.goToPreviousStep(), true
	case isTextStep(m.Step):
		return nil, false
	case keyMsg.Type == tea.KeyLeft:
		return m.goToPreviousStep(), true
	case keyMsg.Type == tea.KeyRight:
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

	if m.Step == StepDockerCheck {
		m.DockerChecking, m.DockerProblem = false, ""
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
	if rebuild := steps[step].rebuild; rebuild != nil {
		return rebuild(m)
	}
	return nil
}
