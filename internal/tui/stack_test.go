package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/vaibdix/django-cli/internal/config"
)

func TestEveryStackReachesTheProjectConfigThroughTheAddOnStep(t *testing.T) {
	for _, stack := range config.AllStacks() {
		var want config.Project
		want.ApplyFeatures(stack.Features())

		m := questionModel(t, StepStack)
		m.SelectedStack = stack
		m.rebuildForm(StepStack).Init()

		m = advance(m, tea.KeyEnter)
		if m.Step != StepProjectConfig {
			t.Fatalf("%s: stack step landed on %v, want StepProjectConfig", stack, m.Step)
		}

		m = advance(m, tea.KeyEnter)

		if m.CreateTemplates != want.CreateTemplates {
			t.Errorf("%s: CreateTemplates = %v after the add-on step, want %v",
				stack, m.CreateTemplates, want.CreateTemplates)
		}
		if m.SetupTailwind != want.SetupTailwind {
			t.Errorf("%s: SetupTailwind = %v, want %v", stack, m.SetupTailwind, want.SetupTailwind)
		}
		if m.SetupHTMX != want.SetupHTMX {
			t.Errorf("%s: SetupHTMX = %v, want %v", stack, m.SetupHTMX, want.SetupHTMX)
		}
	}
}

func TestPickingAnAddOnDoesNotDiscardTheStack(t *testing.T) {
	m := questionModel(t, StepStack)
	m.SelectedStack = config.StackTailwindHTMX
	m.rebuildForm(StepStack).Init()

	m = advance(m, tea.KeyEnter)
	m.SelectedOptions = []config.Feature{config.FeatureRestFramework, config.FeatureRuff}
	m.rebuildForm(StepProjectConfig).Init()
	m = advance(m, tea.KeyEnter)

	if !m.CreateTemplates || !m.SetupTailwind || !m.SetupHTMX {
		t.Errorf("choosing add-ons dropped the stack: templates=%v tailwind=%v htmx=%v",
			m.CreateTemplates, m.SetupTailwind, m.SetupHTMX)
	}
	if !m.SetupRestFramework || !m.SetupRuff {
		t.Errorf("the add-ons themselves were lost: rest=%v ruff=%v", m.SetupRestFramework, m.SetupRuff)
	}
}

func TestReturningToTheStackStepKeepsTheChoice(t *testing.T) {
	m := questionModel(t, StepProjectConfig)
	m.SelectedStack = config.StackTailwind

	m = press(m, tea.KeyLeft)
	if m.Step != StepStack {
		t.Fatalf("left from the add-ons landed on %v, want StepStack", m.Step)
	}
	if m.SelectedStack != config.StackTailwind {
		t.Fatalf("going back reset the stack to %q", m.SelectedStack)
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

func view(m *Model) string { return m.StackForm.View() }

func TestTheStackIsRecordedInTheRunSummary(t *testing.T) {
	m := questionModel(t, StepProjectConfig)
	m.SelectedStack = config.StackHTMX
	m.ProcessFormData()

	summary := strings.Join(m.StepMessages, "\n")
	if !strings.Contains(summary, "Stack: "+config.StackHTMX.Label()) {
		t.Errorf("the run summary does not say which stack was chosen:\n%s", summary)
	}
}
