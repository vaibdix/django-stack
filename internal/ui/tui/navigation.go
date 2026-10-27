package tui

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
)

func (m *Model) handleStepKeys(msg tea.Msg) ([]tea.Cmd, bool) {
	keyMsg, ok := msg.(tea.KeyPressMsg)
	if !ok || !isQuestionStep(m.step) {
		return nil, false
	}

	switch key := keyMsg.String(); {
	case key == "shift+tab":
		return m.goToPreviousStep(), true
	case isTextStep(m.step):
		return nil, false
	case key == "left":
		return m.goToPreviousStep(), true
	case key == "right":
		return m.updateActiveForm(tea.KeyPressMsg{Code: tea.KeyEnter}), true
	default:
		return nil, false
	}
}

func (m *Model) goToPreviousStep() []tea.Cmd {
	target, ok := previousStep(m.step)
	if !ok {
		return nil
	}

	if m.step == stepDockerCheck {
		m.dockerChecking, m.dockerProblem = false, ""
	}

	form := m.rebuildForm(target)
	m.step = target
	m.syncFormWidths()

	if form == nil {
		return nil
	}
	return []tea.Cmd{form.Init()}
}

func (m *Model) rebuildForm(step wizardStep) *huh.Form {
	if rebuild := steps()[step].rebuild; rebuild != nil {
		return rebuild(m)
	}
	return nil
}
