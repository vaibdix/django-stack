package tui

import (
	"testing"

	"charm.land/huh/v2"

	"github.com/vaibdix/django-stack/internal/config"
)

func newStubForm() *huh.Form {
	var value string
	return huh.NewForm(huh.NewGroup(huh.NewInput().Value(&value)))
}

func TestCalculateTotalStepsGrowsWithEnabledFeatures(t *testing.T) {
	t.Parallel()

	bare := (&Model{choices: config.Choices{}}).calculateTotalSteps()

	richer := []struct {
		name string
		cfg  config.Choices
	}{
		{"templates", config.Choices{Stack: config.StackStandard}},
		{"app", config.Choices{App: "blog"}},
		{"ruff", config.Choices{AddOns: config.NewSet(config.Ruff)}},
		{"pre-commit", config.Choices{AddOns: config.NewSet(config.PreCommit)}},
		{"ruff and pre-commit", config.Choices{AddOns: config.NewSet(config.Ruff, config.PreCommit)}},
	}

	for _, tc := range richer {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := (&Model{choices: tc.cfg}).calculateTotalSteps()
			if got <= bare {
				t.Errorf("enabling %s gives %d steps, not more than the bare %d", tc.name, got, bare)
			}
		})
	}
}

func TestCalculateTotalStepsIsAlwaysPositive(t *testing.T) {
	t.Parallel()
	m := &Model{}
	if got := m.calculateTotalSteps(); got <= 0 {
		t.Fatalf("CalculateTotalSteps() = %d on an empty config, want > 0 to avoid divide-by-zero in advanceProgress", got)
	}
}

func TestGetActiveFormReturnsNilForNonFormSteps(t *testing.T) {
	t.Parallel()

	nonFormSteps := []wizardStep{
		stepSplashScreen,
		stepSetup,
		stepComplete,
	}

	for _, step := range nonFormSteps {
		m := &Model{step: step}
		if form := m.currentForm(); form != nil {
			t.Errorf("GetActiveForm() at step %d returned a form, want nil", int(step))
		}
	}
}

func TestGetActiveFormMapsEachStepToItsOwnForm(t *testing.T) {
	t.Parallel()

	m := &Model{
		projectNameForm:   newStubForm(),
		pythonForm:        newStubForm(),
		djangoVersionForm: newStubForm(),
		databaseForm:      newStubForm(),
		projectConfigForm: newStubForm(),
		appNameForm:       newStubForm(),
		devServerForm:     newStubForm(),
	}

	formSteps := []wizardStep{
		stepProjectName,
		stepPythonSelection,
		stepDjangoVersion,
		stepDatabase,
		stepProjectConfig,
		stepAppName,
		stepDevServerPrompt,
	}

	seen := make(map[any]wizardStep, len(formSteps))
	for _, step := range formSteps {
		m.step = step
		form := m.currentForm()
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
	if err := validateAppName("my_shop", "my-shop", config.PythonNames{}); err == nil {
		t.Error("an app named after the project package was accepted; startapp would fail after the install")
	}
	if err := validateAppName("catalog", "my-shop", config.PythonNames{}); err != nil {
		t.Errorf("a valid app name was rejected: %v", err)
	}
}
