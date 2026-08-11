package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"

	"github.com/vaibdix/django-cli/internal/config"
)

func questionModel(t *testing.T, step Step) *Model {
	t.Helper()

	m := NewModel()
	m.Step = step
	m.Width, m.Height = 100, 32
	m.syncFormWidths()
	if form := m.GetActiveForm(); form != nil {
		form.Init()
	}
	return m
}

func press(m *Model, key tea.KeyType) *Model {
	updated, _ := m.Update(tea.KeyMsg{Type: key})
	return updated.(*Model)
}

func TestLeftArrowReturnsToThePreviousQuestion(t *testing.T) {
	for step, want := range previousStep {
		m := questionModel(t, step)

		if got := press(m, tea.KeyLeft).Step; got != want {
			t.Errorf("left from %v landed on %v, want %v", step, got, want)
		}
	}
}

func TestGoingBackDoesNotImmediatelyBounceForwardAgain(t *testing.T) {
	m := questionModel(t, StepDatabase)
	m.DjangoVersionForm.State = huh.StateCompleted

	m = press(m, tea.KeyLeft)
	if m.Step != StepDjangoVersion {
		t.Fatalf("left from the database step landed on %v, want StepDjangoVersion", m.Step)
	}

	updated, _ := m.Update(BlinkMsg{})
	if got := updated.(*Model).Step; got != StepDjangoVersion {
		t.Fatalf("an unrelated message after going back advanced the step to %v; "+
			"the form we returned to is still completed, so it re-submits itself", got)
	}
}

func TestGoingBackKeepsTheAnswerAlreadyGiven(t *testing.T) {
	m := questionModel(t, StepProjectConfig)
	m.DatabaseType = config.MySQLDocker

	m = press(m, tea.KeyLeft)

	if m.DatabaseType != config.MySQLDocker {
		t.Fatalf("returning to the database step reset the answer to %q, want %q",
			m.DatabaseType, config.MySQLDocker)
	}
	if view := m.DatabaseForm.View(); !strings.Contains(view, "MySQL") {
		t.Fatalf("the rebuilt database form does not render its options:\n%s", view)
	}
}

func TestGoingBackKeepsTheFeaturesAlreadyPicked(t *testing.T) {
	m := questionModel(t, StepAppName)
	m.SelectedOptions = []config.Feature{config.FeatureTailwind, config.FeatureHTMX}

	m = press(m, tea.KeyLeft)
	if m.Step != StepProjectConfig {
		t.Fatalf("left from the app name landed on %v, want StepProjectConfig", m.Step)
	}

	view := m.ProjectConfigForm.View()
	for _, line := range strings.Split(view, "\n") {
		if strings.Contains(line, "Tailwind CSS v4") && !strings.Contains(line, "✓") {
			t.Fatalf("Tailwind was picked but the rebuilt feature list shows it unselected:\n%s", view)
		}
	}
}

func TestLeftArrowOnTheFirstQuestionStaysPut(t *testing.T) {
	m := questionModel(t, StepProjectName)

	if got := press(m, tea.KeyLeft).Step; got != StepProjectName {
		t.Fatalf("left from the first question moved to %v; there is nothing before it", got)
	}
}

func settle(m *Model, cmd tea.Cmd) *Model {
	queue := []tea.Cmd{cmd}

	for round := 0; len(queue) > 0 && round < 50; round++ {
		next := queue[0]
		queue = queue[1:]
		if next == nil || m.Step == StepSetup {
			continue
		}

		msg := next()
		if batch, ok := msg.(tea.BatchMsg); ok {
			queue = append(queue, batch...)
			continue
		}
		if _, ok := msg.(spinner.TickMsg); ok || msg == nil {
			continue
		}

		updated, cmd := m.Update(msg)
		m = updated.(*Model)
		queue = append(queue, cmd)
	}

	return m
}

func advance(m *Model, key tea.KeyType) *Model {
	updated, cmd := m.Update(tea.KeyMsg{Type: key})
	return settle(updated.(*Model), cmd)
}

func TestRightArrowAdvancesLikeEnter(t *testing.T) {
	withRight := advance(questionModel(t, StepDatabase), tea.KeyRight)
	withEnter := advance(questionModel(t, StepDatabase), tea.KeyEnter)

	if withRight.Step != withEnter.Step {
		t.Fatalf("right arrow reached %v where enter reached %v", withRight.Step, withEnter.Step)
	}
	if withRight.Step == StepDatabase {
		t.Fatal("neither key advanced past the database step, so this proves nothing")
	}
}

func TestRightArrowConfirmsTheFeatureListInsteadOfSkippingIt(t *testing.T) {
	m := questionModel(t, StepProjectConfig)
	m.SelectedOptions = []config.Feature{config.FeatureRestFramework}
	m.rebuildForm(StepProjectConfig).Init()

	m = advance(m, tea.KeyRight)

	if m.Step != StepAppName {
		t.Fatalf("right from the feature list landed on %v, want StepAppName", m.Step)
	}
	if !m.SetupRestFramework {
		t.Fatal("right advanced without applying the picked features")
	}
}

func TestArrowsDoNotHijackTheConfirmPrompt(t *testing.T) {
	m := questionModel(t, StepDevServerPrompt)
	before := m.StartDevServer

	m = press(m, tea.KeyRight)

	if m.Step != StepDevServerPrompt {
		t.Fatalf("an arrow navigated away from the confirm prompt to %v", m.Step)
	}
	if m.StartDevServer == before {
		t.Fatal("right no longer toggles the confirm prompt; step navigation swallowed it")
	}
}

func TestArrowsAreIgnoredWhileTheProjectIsBeingBuilt(t *testing.T) {
	m := questionModel(t, StepSetup)

	if got := press(m, tea.KeyLeft).Step; got != StepSetup {
		t.Fatalf("left during generation moved to %v; the scaffold is already running", got)
	}
}

func TestEveryBackStepIsTheImmediatelyPrecedingScreen(t *testing.T) {
	for step, target := range previousStep {
		here, ok := stepPositions[step]
		if !ok {
			t.Errorf("%v can go back but has no position in the header counter", step)
			continue
		}
		there, ok := stepPositions[target]
		if !ok {
			t.Errorf("%v is a back target but has no position in the header counter", target)
			continue
		}
		if there != here-1 {
			t.Errorf("left from step %d goes to step %d; it must go to step %d", here, there, here-1)
		}
	}
}

func TestEveryQuestionAfterTheFirstCanGoBack(t *testing.T) {
	for step := range stepPositions {
		if !isQuestionStep(step) || step == StepProjectName {
			continue
		}
		if _, ok := previousStep[step]; !ok {
			t.Errorf("%v asks the user a question but has no back target, so left is a dead key there", step)
		}
	}
}

func TestRebuildFormProducesTheFormThatStepUses(t *testing.T) {
	for step := range previousStep {
		m := NewModel()
		m.Step = step

		rebuilt := m.rebuildForm(step)
		if rebuilt == nil {
			t.Errorf("rebuildForm(%v) returned nil", step)
			continue
		}
		if rebuilt != m.GetActiveForm() {
			t.Errorf("rebuildForm(%v) did not store the form where GetActiveForm looks for it", step)
		}
	}
}

func TestReturningToTheFeatureListDoesNotDuplicateTheSummary(t *testing.T) {
	m := questionModel(t, StepProjectConfig)
	m.SelectedOptions = []config.Feature{config.FeatureTailwind}

	m.ProcessFormData()
	first := len(m.StepMessages)

	m.ProcessFormData()
	if got := len(m.StepMessages); got != first {
		t.Fatalf("the configuration summary grew from %d to %d lines after going back and forward again", first, got)
	}
}
