package tui

import (
	"fmt"
	"strings"
	"testing"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"

	"github.com/vaibdix/django-stack/internal/config"
	"github.com/vaibdix/django-stack/internal/scaffold"
)

func questionModel(t *testing.T, step wizardStep) *Model {
	t.Helper()

	m := NewModel(scaffold.Options{})
	m.step = step
	m.width, m.height = 100, 32
	m.syncFormWidths()
	if form := m.currentForm(); form != nil {
		form.Init()
	}
	return m
}

type unrelatedMsg struct{}

var (
	enterKey    = tea.KeyPressMsg{Code: tea.KeyEnter}
	leftKey     = tea.KeyPressMsg{Code: tea.KeyLeft}
	rightKey    = tea.KeyPressMsg{Code: tea.KeyRight}
	downKey     = tea.KeyPressMsg{Code: tea.KeyDown}
	escKey      = tea.KeyPressMsg{Code: tea.KeyEscape}
	spaceKey    = tea.KeyPressMsg{Code: tea.KeySpace, Text: " "}
	shiftTabKey = tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift}
	ctrlCKey    = tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}
)

func runeKey(r rune) tea.KeyPressMsg { return tea.KeyPressMsg{Code: r, Text: string(r)} }

func mustModel(model tea.Model) *Model {
	m, ok := model.(*Model)
	if !ok {
		panic(fmt.Sprintf("Update returned %T, want *Model", model))
	}
	return m
}

func press(m *Model, key tea.KeyPressMsg) *Model {
	updated, _ := m.Update(key)
	return mustModel(updated)
}

func TestShiftTabReturnsToThePreviousQuestionFromEveryScreen(t *testing.T) {
	for step, want := range backTargets() {
		m := questionModel(t, step)

		if got := press(m, shiftTabKey).step; got != want {
			t.Errorf("shift+tab from %v landed on %v, want %v", step, got, want)
		}
	}
}

func TestLeftArrowReturnsToThePreviousQuestionFromSelects(t *testing.T) {
	for step, want := range backTargets() {
		if isTextStep(step) {
			continue
		}
		m := questionModel(t, step)

		if got := press(m, leftKey).step; got != want {
			t.Errorf("left from %v landed on %v, want %v", step, got, want)
		}
	}
}

func TestArrowsMoveTheCursorInsideTextFields(t *testing.T) {
	for _, step := range []wizardStep{stepProjectName, stepAppName} {
		m := questionModel(t, step)
		m = typeRunes(m, "blg")
		m = press(m, leftKey)
		m = typeRunes(m, "o")
		m = press(m, rightKey)

		if m.step != step {
			t.Fatalf("an arrow key left the %v text field for %v", step, m.step)
		}
		value := m.project.ProjectName
		if step == stepAppName {
			value = m.project.AppName
		}
		if !strings.HasSuffix(value, "blog") {
			t.Errorf("left did not move the cursor in %v: value is %q", step, value)
		}
	}
}

func TestAnyKeySkipsTheSplash(t *testing.T) {
	m := NewModel(scaffold.Options{})
	m.width, m.height = 100, 32

	if got := press(m, spaceKey).step; got != stepProjectName {
		t.Errorf("a key on the splash landed on %v, want the project name", got)
	}
}

func TestGoingBackDoesNotImmediatelyBounceForwardAgain(t *testing.T) {
	m := questionModel(t, stepDatabase)
	m.djangoVersionForm.State = huh.StateCompleted

	m = press(m, leftKey)
	if m.step != stepDjangoVersion {
		t.Fatalf("left from the database step landed on %v, want StepDjangoVersion", m.step)
	}

	updated, _ := m.Update(unrelatedMsg{})
	if got := mustModel(updated).step; got != stepDjangoVersion {
		t.Fatalf("an unrelated message after going back advanced the step to %v; "+
			"the form we returned to is still completed, so it re-submits itself", got)
	}
}

func TestGoingBackKeepsTheAnswerAlreadyGiven(t *testing.T) {
	m := questionModel(t, stepStack)
	m.project.DatabaseType = config.MySQLDocker

	m = press(m, leftKey)

	if m.project.DatabaseType != config.MySQLDocker {
		t.Fatalf("returning to the database step reset the answer to %q, want %q",
			m.project.DatabaseType, config.MySQLDocker)
	}
	if view := m.databaseForm.View(); !strings.Contains(view, "MySQL") {
		t.Fatalf("the rebuilt database form does not render its options:\n%s", view)
	}
}

func TestGoingBackKeepsTheFeaturesAlreadyPicked(t *testing.T) {
	m := questionModel(t, stepAppName)
	m.selectedOptions = []config.Feature{config.FeatureRestFramework, config.FeatureRuff}

	m = press(m, shiftTabKey)
	if m.step != stepProjectConfig {
		t.Fatalf("shift+tab from the app name landed on %v, want StepProjectConfig", m.step)
	}

	view := m.projectConfigForm.View()
	for _, line := range strings.Split(view, "\n") {
		if strings.Contains(line, "Django REST Framework") && !strings.Contains(line, "✓") {
			t.Fatalf("REST Framework was picked but the rebuilt add-on list shows it unselected:\n%s", view)
		}
	}
}

func TestLeftArrowOnTheFirstQuestionStaysPut(t *testing.T) {
	m := questionModel(t, stepProjectName)

	if got := press(m, leftKey).step; got != stepProjectName {
		t.Fatalf("left from the first question moved to %v; there is nothing before it", got)
	}
}

func settle(m *Model, cmd tea.Cmd) *Model {
	queue := []tea.Cmd{cmd}

	for round := 0; len(queue) > 0 && round < 50; round++ {
		next := queue[0]
		queue = queue[1:]
		if next == nil || m.step == stepSetup {
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
		m = mustModel(updated)
		queue = append(queue, cmd)
	}

	return m
}

func advance(m *Model, key tea.KeyPressMsg) *Model {
	updated, cmd := m.Update(key)
	return settle(mustModel(updated), cmd)
}

func TestRightArrowAdvancesLikeEnter(t *testing.T) {
	withRight := advance(questionModel(t, stepDatabase), rightKey)
	withEnter := advance(questionModel(t, stepDatabase), enterKey)

	if withRight.step != withEnter.step {
		t.Fatalf("right arrow reached %v where enter reached %v", withRight.step, withEnter.step)
	}
	if withRight.step == stepDatabase {
		t.Fatal("neither key advanced past the database step, so this proves nothing")
	}
}

func TestRightArrowConfirmsTheFeatureListInsteadOfSkippingIt(t *testing.T) {
	m := questionModel(t, stepProjectConfig)
	m.selectedOptions = []config.Feature{config.FeatureRestFramework}
	m.rebuildForm(stepProjectConfig).Init()

	m = advance(m, rightKey)

	if m.step != stepAppName {
		t.Fatalf("right from the feature list landed on %v, want StepAppName", m.step)
	}
	if !m.project.SetupRestFramework {
		t.Fatal("right advanced without applying the picked features")
	}
}

func TestArrowsDoNotHijackTheConfirmPrompt(t *testing.T) {
	m := questionModel(t, stepDevServerPrompt)
	before := m.startDevServer

	m = press(m, rightKey)

	if m.step != stepDevServerPrompt {
		t.Fatalf("an arrow navigated away from the confirm prompt to %v", m.step)
	}
	if m.startDevServer == before {
		t.Fatal("right no longer toggles the confirm prompt; step navigation swallowed it")
	}
}

func TestArrowsAreIgnoredWhileTheProjectIsBeingBuilt(t *testing.T) {
	m := questionModel(t, stepSetup)

	if got := press(m, leftKey).step; got != stepSetup {
		t.Fatalf("left during generation moved to %v; the scaffold is already running", got)
	}
}

func TestEveryBackStepIsTheImmediatelyPrecedingScreen(t *testing.T) {
	for step, target := range backTargets() {
		here, ok := stepPosition(step)
		if !ok {
			t.Errorf("%v can go back but has no position in the header counter", step)
			continue
		}
		there, ok := stepPosition(target)
		if !ok {
			t.Errorf("%v is a back target but has no position in the header counter", target)
			continue
		}
		if there != here && there != here-1 {
			t.Errorf("left from step %d goes to step %d; it must stay on the same numbered step "+
				"(a gate like the Docker check) or move back exactly one", here, there)
		}
	}
}

func TestEveryQuestionAfterTheFirstCanGoBack(t *testing.T) {
	for step := range steps() {
		if _, numbered := stepPosition(step); !numbered || !isQuestionStep(step) || step == stepProjectName {
			continue
		}
		if _, ok := previousStep(step); !ok {
			t.Errorf("%v asks the user a question but has no back target, so left is a dead key there", step)
		}
	}
}

func TestEveryBackTargetCanBeRebuilt(t *testing.T) {
	for from, target := range backTargets() {
		m := NewModel(scaffold.Options{})
		m.step = target

		rebuilt := m.rebuildForm(target)
		if rebuilt == nil {
			t.Errorf("left from %v lands on %v, but rebuildForm(%v) returned nil, "+
				"so the step would show a stale completed form", from, target, target)
			continue
		}
		if rebuilt != m.currentForm() {
			t.Errorf("rebuildForm(%v) did not store the form where GetActiveForm looks for it", target)
		}
	}
}

func TestReturningToTheFeatureListDoesNotDuplicateTheSummary(t *testing.T) {
	m := questionModel(t, stepProjectConfig)
	m.selectedOptions = []config.Feature{config.FeatureTailwind}

	m.processFormData()
	first := len(m.stepMessages)

	m.processFormData()
	if got := len(m.stepMessages); got != first {
		t.Fatalf("the configuration summary grew from %d to %d lines after going back and forward again", first, got)
	}
}

func TestEveryStepIsDefinedOnceInTheTable(t *testing.T) {
	t.Parallel()

	for step := stepSplashScreen; step <= stepComplete; step++ {
		spec, ok := steps()[step]
		if !ok {
			t.Errorf("step %d has no entry in the steps table", step)
			continue
		}
		if spec.title == "" {
			t.Errorf("step %d has no title", step)
		}
		if spec.question && (spec.form == nil || spec.next == nil) {
			t.Errorf("question step %q has no form or no next", spec.title)
		}
		if spec.hasPrevious && steps()[spec.previous].rebuild == nil {
			t.Errorf("going back from %q lands on %q, which cannot rebuild its form", spec.title, steps()[spec.previous].title)
		}
	}
}

func backTargets() map[wizardStep]wizardStep {
	targets := map[wizardStep]wizardStep{}
	for step := range steps() {
		if target, ok := previousStep(step); ok {
			targets[step] = target
		}
	}
	return targets
}
