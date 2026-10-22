package tui

import (
	"strings"
	"testing"

	"github.com/vaibdix/django-stack/internal/config"
)

func TestEveryStackReachesTheProjectConfigThroughTheAddOnStep(t *testing.T) {
	for _, stack := range config.AllStacks() {
		var want config.Project
		want.ApplyFeatures(stack.Features())

		m := questionModel(t, stepStack)
		m.selectedStack = stack
		m.rebuildForm(stepStack).Init()

		m = advance(m, enterKey)
		if m.step != stepProjectConfig {
			t.Fatalf("%s: stack step landed on %v, want StepProjectConfig", stack, m.step)
		}

		m = advance(m, enterKey)

		if m.project.CreateTemplates != want.CreateTemplates {
			t.Errorf("%s: CreateTemplates = %v after the add-on step, want %v",
				stack, m.project.CreateTemplates, want.CreateTemplates)
		}
		if m.project.SetupTailwind != want.SetupTailwind {
			t.Errorf("%s: SetupTailwind = %v, want %v", stack, m.project.SetupTailwind, want.SetupTailwind)
		}
		if m.project.SetupHTMX != want.SetupHTMX {
			t.Errorf("%s: SetupHTMX = %v, want %v", stack, m.project.SetupHTMX, want.SetupHTMX)
		}
	}
}

func TestPickingAnAddOnDoesNotDiscardTheStack(t *testing.T) {
	m := questionModel(t, stepStack)
	m.selectedStack = config.StackTailwindHTMX
	m.rebuildForm(stepStack).Init()

	m = advance(m, enterKey)
	m.selectedOptions = []config.Feature{config.FeatureRestFramework, config.FeatureRuff}
	m.rebuildForm(stepProjectConfig).Init()
	m = advance(m, enterKey)

	if !m.project.CreateTemplates || !m.project.SetupTailwind || !m.project.SetupHTMX {
		t.Errorf("choosing add-ons dropped the stack: templates=%v tailwind=%v htmx=%v",
			m.project.CreateTemplates, m.project.SetupTailwind, m.project.SetupHTMX)
	}
	if !m.project.SetupRestFramework || !m.project.SetupRuff {
		t.Errorf("the add-ons themselves were lost: rest=%v ruff=%v", m.project.SetupRestFramework, m.project.SetupRuff)
	}
}

func TestReturningToTheStackStepKeepsTheChoice(t *testing.T) {
	m := questionModel(t, stepProjectConfig)
	m.selectedStack = config.StackTailwind

	m = press(m, leftKey)
	if m.step != stepStack {
		t.Fatalf("left from the add-ons landed on %v, want StepStack", m.step)
	}
	if m.selectedStack != config.StackTailwind {
		t.Fatalf("going back reset the stack to %q", m.selectedStack)
	}

	var marked bool
	for _, line := range strings.Split(view(m), "\n") {
		fields := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "┃"))
		if fields == "> "+config.StackTailwind.Label() {
			marked = true
		}
	}
	if !marked {
		t.Errorf("the rebuilt stack form does not show the chosen option as selected:\n%s", view(m))
	}
}

func view(m *Model) string { return plain(m.stackForm.View()) }

func TestTheStackIsRecordedInTheRunSummary(t *testing.T) {
	m := questionModel(t, stepProjectConfig)
	m.selectedStack = config.StackHTMX
	m.processFormData()

	summary := strings.Join(m.stepMessages, "\n")
	if !strings.Contains(summary, "Stack: "+config.StackHTMX.Label()) {
		t.Errorf("the run summary does not say which stack was chosen:\n%s", summary)
	}
}
