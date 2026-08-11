package tui

import (
	"testing"

	"github.com/charmbracelet/huh"

	"github.com/vaibdix/django-cli/internal/config"
)

// newStubForm builds a minimal, distinct *huh.Form for form-routing assertions.
func newStubForm() *huh.Form {
	var value string
	return huh.NewForm(huh.NewGroup(huh.NewInput().Value(&value)))
}

// These tests construct Model literals directly rather than calling NewModel().
// NewModel() shells out to discover Python interpreters and queries PyPI for
// Django releases, so it is unusable in a hermetic unit test.

// The relative behaviour of the step count is what matters — enabling a feature
// must add work, and the absolute numbers belong to the plan, not to the TUI.
//
// This replaces a table that asserted specific totals (3 for a bare project, 4
// with templates, ...). Those numbers were the old hand-maintained accounting
// and they were wrong: a minimal project runs twelve steps, not three, which is
// exactly why the bar used to overshoot. Pinning absolute counts here would
// only re-encode that mistake in a different place — the exact figures are
// asserted in internal/scaffold against the plan itself.
func TestCalculateTotalStepsGrowsWithEnabledFeatures(t *testing.T) {
	t.Parallel()

	bare := (&Model{Project: config.Project{}}).CalculateTotalSteps()

	richer := []struct {
		name string
		cfg  config.Project
	}{
		{"templates", config.Project{CreateTemplates: true}},
		{"app", config.Project{AppName: "blog"}},
		{"ruff", config.Project{SetupRuff: true}},
		{"pre-commit", config.Project{SetupPreCommit: true}},
		{"ruff and pre-commit", config.Project{SetupRuff: true, SetupPreCommit: true}},
	}

	for _, tc := range richer {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := (&Model{Project: tc.cfg}).CalculateTotalSteps()
			if got <= bare {
				t.Errorf("enabling %s gives %d steps, not more than the bare %d", tc.name, got, bare)
			}
		})
	}
}

func TestCalculateTotalStepsIsAlwaysPositive(t *testing.T) {
	t.Parallel()

	// advanceProgress divides by TotalSteps, so a zero result would produce
	// +Inf / NaN progress values.
	m := &Model{}
	if got := m.CalculateTotalSteps(); got <= 0 {
		t.Fatalf("CalculateTotalSteps() = %d on an empty config, want > 0 to avoid divide-by-zero in advanceProgress", got)
	}
}

func TestGetActiveFormReturnsNilForNonFormSteps(t *testing.T) {
	t.Parallel()

	nonFormSteps := []Step{
		StepSplashScreen,
		StepSetup,
		StepComplete,
	}

	for _, step := range nonFormSteps {
		m := &Model{Step: step}
		if form := m.GetActiveForm(); form != nil {
			t.Errorf("GetActiveForm() at step %d returned a form, want nil", int(step))
		}
	}
}

// Every step that GetActiveForm claims to serve must map to a distinct field.
// A copy-paste error in the switch would otherwise route two steps to the same
// form and silently skip a screen.
func TestGetActiveFormMapsEachStepToItsOwnForm(t *testing.T) {
	t.Parallel()

	m := &Model{
		ProjectNameForm:   newStubForm(),
		PythonForm:        newStubForm(),
		DjangoVersionForm: newStubForm(),
		DatabaseForm:      newStubForm(),
		ProjectConfigForm: newStubForm(),
		AppNameForm:       newStubForm(),
		DevServerForm:     newStubForm(),
	}

	formSteps := []Step{
		StepProjectName,
		StepPythonSelection,
		StepDjangoVersion,
		StepDatabase,
		StepProjectConfig,
		StepAppName,
		StepDevServerPrompt,
	}

	seen := make(map[any]Step, len(formSteps))
	for _, step := range formSteps {
		m.Step = step
		form := m.GetActiveForm()
		if form == nil {
			t.Errorf("GetActiveForm() at step %d = nil, want a form", int(step))
			continue
		}
		if prev, dup := seen[form]; dup {
			t.Errorf("steps %d and %d both resolve to the same form", int(prev), int(step))
		}
		seen[form] = step
	}
}
