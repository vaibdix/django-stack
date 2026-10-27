package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func typeRunes(m *Model, text string) *Model {
	for _, r := range text {
		updated, _ := m.Update(runeKey(r))
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
	t.Parallel()

	for _, step := range []wizardStep{stepProjectName, stepAppName} {
		t.Run(screenTitle(step), func(t *testing.T) {
			t.Parallel()

			m := questionModel(t, step)

			m = typeRunes(m, "qqq")

			if m.step != step {
				t.Fatalf("typing landed on %v; the run was torn down mid-question", m.step)
			}

			view := m.render()
			if !strings.Contains(view, "qqq") {
				t.Errorf("the letters never reached the input, so %v cannot accept a name "+
					"containing them:\n%s", step, view)
			}
		})
	}
}

func TestEveryQuestionLetsQuitBeTypedInsteadOfQuitting(t *testing.T) {
	t.Parallel()

	for step := range backTargets() {
		m := questionModel(t, step)

		if quits(m, runeKey('q')) {
			t.Errorf("q quit the run on %v; it must reach the form so names may contain it", step)
		}
	}
}

func TestCtrlCQuitsFromEveryScreen(t *testing.T) {
	t.Parallel()

	for _, step := range []wizardStep{
		stepSplashScreen, stepProjectName, stepPythonSelection, stepDjangoVersion,
		stepDatabase, stepDockerCheck, stepStack, stepProjectConfig, stepAppName,
		stepSetup, stepDevServerPrompt, stepComplete,
	} {
		m := questionModel(t, step)

		if !quits(m, ctrlCKey) {
			t.Errorf("ctrl+c did not quit on %v", step)
		}
	}
}

func TestTheTerminalScreensStillExitOnASingleKey(t *testing.T) {
	t.Parallel()

	for _, key := range []tea.KeyPressMsg{runeKey('q'), enterKey, escKey} {
		m := questionModel(t, stepComplete)

		if !quits(m, key) {
			t.Errorf("%q did not exit the finished run; the footer offers it as an exit", key.String())
		}
	}
}

func TestTheFooterOffersQuitOnlyWhereItWorks(t *testing.T) {
	t.Parallel()

	for step := range steps() {
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
