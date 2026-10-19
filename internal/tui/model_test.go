package tui

import (
	"testing"

	"github.com/charmbracelet/huh"

	"github.com/vaibdix/django-cli/internal/config"
)

func newStubForm() *huh.Form {
	var value string
	return huh.NewForm(huh.NewGroup(huh.NewInput().Value(&value)))
}

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

func TestTheAppNameFormRejectsTheProjectsOwnPackage(t *testing.T) {
	if err := validateAppName("my_shop", "my-shop"); err == nil {
		t.Error("an app named after the project package was accepted; startapp would fail after the install")
	}
	if err := validateAppName("catalog", "my-shop"); err != nil {
		t.Errorf("a valid app name was rejected: %v", err)
	}
}
