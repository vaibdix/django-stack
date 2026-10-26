package tui

import (
	"strings"
	"testing"

	"github.com/vaibdix/django-stack/internal/config"
)

func TestEveryStackReachesTheProjectConfigThroughTheAddOnStep(t *testing.T) {
	t.Parallel()

	for _, stack := range config.AllStacks() {
		var want config.Choices
		want.Stack, want.AddOns = stack, nil

		m := questionModel(t, stepStack)
		m.choices.Stack = stack
		m.rebuildForm(stepStack).Init()

		m = advance(m, enterKey)
		if m.step != stepProjectConfig {
			t.Fatalf("%s: stack step landed on %v, want StepProjectConfig", stack, m.step)
		}

		m = advance(m, enterKey)

		if m.choices.CreateTemplates() != want.CreateTemplates() {
			t.Errorf("%s: CreateTemplates = %v after the add-on step, want %v",
				stack, m.choices.CreateTemplates(), want.CreateTemplates())
		}
		if m.choices.UsesTailwind() != want.UsesTailwind() {
			t.Errorf("%s: SetupTailwind = %v, want %v", stack, m.choices.UsesTailwind(), want.UsesTailwind())
		}
		if m.choices.UsesHTMX() != want.UsesHTMX() {
			t.Errorf("%s: SetupHTMX = %v, want %v", stack, m.choices.UsesHTMX(), want.UsesHTMX())
		}
	}
}

func TestPickingAnAddOnDoesNotDiscardTheStack(t *testing.T) {
	t.Parallel()

	m := questionModel(t, stepStack)
	m.choices.Stack = config.StackTailwindHTMX
	m.rebuildForm(stepStack).Init()

	m = advance(m, enterKey)
	m.selectedOptions = []config.AddOn{config.RestFramework, config.Ruff}
	m.rebuildForm(stepProjectConfig).Init()
	m = advance(m, enterKey)

	if !m.choices.CreateTemplates() || !m.choices.UsesTailwind() || !m.choices.UsesHTMX() {
		t.Errorf("choosing add-ons dropped the stack: templates=%v tailwind=%v htmx=%v",
			m.choices.CreateTemplates(), m.choices.UsesTailwind(), m.choices.UsesHTMX())
	}
	if !m.choices.Has(config.RestFramework) || !m.choices.Has(config.Ruff) {
		t.Errorf("the add-ons themselves were lost: rest=%v ruff=%v", m.choices.Has(config.RestFramework), m.choices.Has(config.Ruff))
	}
}

func TestReturningToTheStackStepKeepsTheChoice(t *testing.T) {
	t.Parallel()

	m := questionModel(t, stepProjectConfig)
	m.choices.Stack = config.StackTailwind

	m = press(m, leftKey)
	if m.step != stepStack {
		t.Fatalf("left from the add-ons landed on %v, want StepStack", m.step)
	}
	if m.choices.Stack != config.StackTailwind {
		t.Fatalf("going back reset the stack to %q", m.choices.Stack)
	}

	var marked bool
	for line := range strings.SplitSeq(view(m), "\n") {
		fields := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "┃"))
		if fields == "> "+config.StackTailwind.Spec().Label {
			marked = true
		}
	}
	if !marked {
		t.Errorf("the rebuilt stack form does not show the chosen option as selected:\n%s", view(m))
	}
}

func view(m *Model) string { return plain(m.stackForm.View()) }

func TestTheStackIsRecordedInTheRunSummary(t *testing.T) {
	t.Parallel()

	m := questionModel(t, stepProjectConfig)
	m.choices.Stack = config.StackHTMX
	m.processFormData()

	summary := strings.Join(m.stepMessages, "\n")
	if !strings.Contains(summary, "Stack: "+config.StackHTMX.Spec().Label) {
		t.Errorf("the run summary does not say which stack was chosen:\n%s", summary)
	}
}

func TestTheSummarySaysWhichAddOnWasAddedAndWhy(t *testing.T) {
	t.Parallel()

	m := questionModel(t, stepProjectConfig)
	m.selectedOptions = []config.AddOn{config.Ruff, config.PreCommit}
	m.processFormData()

	summary := strings.Join(m.stepMessages, "\n")
	for _, want := range []string{"Add-ons: ruff, pre-commit", "Added git, which pre-commit needs"} {
		if !strings.Contains(summary, want) {
			t.Errorf("the run summary is missing %q:\n%s", want, summary)
		}
	}
	if !m.choices.Has(config.Git) {
		t.Error("pre-commit was chosen without git, and git was not added")
	}
}
