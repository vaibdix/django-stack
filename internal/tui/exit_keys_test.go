package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func typeRunes(m *Model, text string) *Model {
	for _, r := range text {
		updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = mustModel(updated)
	}
	return m
}

func quits(m *Model, msg tea.Msg) bool {
	_, cmd := m.Update(msg)
	if cmd == nil {
		return false
	}
	return isQuitCmd(cmd)
}

func isQuitCmd(cmd tea.Cmd) bool {
	switch cmd().(type) {
	case tea.QuitMsg:
		return true
	case tea.BatchMsg:
		return false
	default:
		return false
	}
}

func TestALetterInATextFieldIsTypedRatherThanTreatedAsQuit(t *testing.T) {
	for _, step := range []Step{StepProjectName, StepAppName} {
		t.Run(screenTitles[step], func(t *testing.T) {
			m := questionModel(t, step)

			m = typeRunes(m, "qqq")

			if m.Step != step {
				t.Fatalf("typing landed on %v; the run was torn down mid-question", m.Step)
			}

			view := m.View()
			if !strings.Contains(view, "qqq") {
				t.Errorf("the letters never reached the input, so %v cannot accept a name "+
					"containing them:\n%s", step, view)
			}
		})
	}
}

func TestEveryQuestionLetsQuitBeTypedInsteadOfQuitting(t *testing.T) {
	for step := range previousStep {
		m := questionModel(t, step)

		if quits(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}) {
			t.Errorf("q quit the run on %v; it must reach the form so names may contain it", step)
		}
	}
}

func TestCtrlCQuitsFromEveryScreen(t *testing.T) {
	for _, step := range []Step{
		StepSplashScreen, StepProjectName, StepPythonSelection, StepDjangoVersion,
		StepDatabase, StepDockerCheck, StepStack, StepProjectConfig, StepAppName,
		StepSetup, StepDevServerPrompt, StepComplete,
	} {
		m := questionModel(t, step)

		if !quits(m, tea.KeyMsg{Type: tea.KeyCtrlC}) {
			t.Errorf("ctrl+c did not quit on %v", step)
		}
	}
}

func TestTheTerminalScreensStillExitOnASingleKey(t *testing.T) {
	for _, key := range []tea.KeyMsg{
		{Type: tea.KeyRunes, Runes: []rune{'q'}},
		{Type: tea.KeyEnter},
		{Type: tea.KeyEsc},
	} {
		m := questionModel(t, StepComplete)

		if !quits(m, key) {
			t.Errorf("%q did not exit the finished run; the footer offers it as an exit", key.String())
		}
	}
}

func TestTheFooterOffersQuitOnlyWhereItWorks(t *testing.T) {
	for step := range screenTitles {
		m := questionModel(t, step)

		for _, key := range m.footerCopy().keys {
			if !strings.HasPrefix(key, "q ") {
				continue
			}
			if !m.atTerminalScreen() {
				t.Errorf("%v advertises %q while a text field may still be focused", step, key)
			}
		}
	}
}
