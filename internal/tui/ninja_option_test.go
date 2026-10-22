package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/huh"

	"github.com/vaibdix/django-stack/internal/config"
)

func TestNinjaIsOfferedForEverySupportedDjango(t *testing.T) {
	t.Parallel()

	offered := false
	for _, option := range projectConfigOptions() {
		offered = offered || option.Value == config.FeatureNinja
	}
	if !offered {
		t.Error("django-ninja is missing from the add-ons; since 1.7.0 it installs on every supported Django")
	}
}

func TestTheTwoAPIFrameworksCannotBothBeChosen(t *testing.T) {
	t.Parallel()

	both := []config.Feature{config.FeatureRestFramework, config.FeatureNinja}
	err := validateOneAPIFramework(both)
	if err == nil {
		t.Fatal("picking both API frameworks was accepted; they both generate api/ and mount /api/")
	}
	if !strings.Contains(err.Error(), "one API framework") {
		t.Errorf("the message does not say to pick one: %q", err)
	}

	for _, selection := range [][]config.Feature{
		{config.FeatureRestFramework},
		{config.FeatureNinja},
		{config.FeatureNinja, config.FeatureMypy, config.FeatureRuff},
		{},
	} {
		if err := validateOneAPIFramework(selection); err != nil {
			t.Errorf("a valid selection %v was rejected: %v", selection, err)
		}
	}
}

func TestDjango61OffersNinja(t *testing.T) {
	t.Parallel()

	m := questionModel(t, StepStack)
	m.DjangoVersion = "6.1.2"
	m.StackForm.State = huh.StateCompleted

	m.advanceFromCompletedForm()
	if m.Step != StepProjectConfig {
		t.Fatalf("advancing from the stack step landed on %v", m.Step)
	}
	if view := m.ProjectConfigForm.View(); !strings.Contains(view, "Django Ninja") {
		t.Errorf("the add-on list hides Django Ninja on 6.1:\n%s", view)
	}
}
