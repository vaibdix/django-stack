package tui

import (
	"strings"
	"testing"

	"charm.land/huh/v2"

	"github.com/vaibdix/django-stack/internal/config"
)

func TestNinjaIsOfferedForEverySupportedDjango(t *testing.T) {
	t.Parallel()

	offered := false
	for _, option := range projectConfigOptions() {
		offered = offered || option.Value == config.Ninja
	}
	if !offered {
		t.Error("django-ninja is missing from the add-ons; since 1.7.0 it installs on every supported Django")
	}
}

func TestDjango61OffersNinja(t *testing.T) {
	t.Parallel()

	m := questionModel(t, stepStack)
	m.project.DjangoVersion = "6.1.2"
	m.stackForm.State = huh.StateCompleted

	m.advanceFromCompletedForm()
	if m.step != stepProjectConfig {
		t.Fatalf("advancing from the stack step landed on %v", m.step)
	}
	if view := m.projectConfigForm.View(); !strings.Contains(view, "Django Ninja") {
		t.Errorf("the add-on list hides Django Ninja on 6.1:\n%s", view)
	}
}
