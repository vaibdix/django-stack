package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/huh"

	"github.com/vaibdix/django-cli/internal/config"
)

func optionValues(djangoVersion string) []config.Feature {
	options := projectConfigOptions(djangoVersion)
	values := make([]config.Feature, 0, len(options))
	for _, o := range options {
		values = append(values, o.Value)
	}
	return values
}

func offersNinja(djangoVersion string) bool {
	for _, v := range optionValues(djangoVersion) {
		if v == config.FeatureNinja {
			return true
		}
	}
	return false
}

func TestTheNinjaOptionIsWithheldFromDjangoThatCannotInstallIt(t *testing.T) {
	t.Parallel()

	for _, v := range []string{"6.1", "6.1.3", "7.0"} {
		if offersNinja(v) {
			t.Errorf("django %s offers django-ninja, but every release pins Django<6.1, "+
				"so the install would fail after the venv is built", v)
		}
	}

	for _, v := range []string{"4.2", "5.2.12", "6.0", "6.0.3"} {
		if !offersNinja(v) {
			t.Errorf("django %s can install django-ninja but the option is missing", v)
		}
	}
}

func TestEveryAddOnStaysReachableWhenNinjaIsWithheld(t *testing.T) {
	t.Parallel()

	withNinja := optionValues("6.0")
	withoutNinja := optionValues("6.1")

	if len(withNinja)-len(withoutNinja) != 1 {
		t.Fatalf("hiding ninja changed the option count by %d, want 1",
			len(withNinja)-len(withoutNinja))
	}

	for _, feature := range withoutNinja {
		if feature == config.FeatureNinja {
			t.Error("ninja is still listed on a Django that cannot install it")
		}
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

func TestTheAddOnFormIsRebuiltOnceTheDjangoVersionIsKnown(t *testing.T) {
	t.Parallel()

	m := questionModel(t, StepStack)
	m.DjangoVersion = "6.1"
	m.StackForm.State = huh.StateCompleted

	cmds := m.advanceFromCompletedForm()
	_ = cmds

	if m.Step != StepProjectConfig {
		t.Fatalf("advancing from the stack step landed on %v", m.Step)
	}

	view := m.ProjectConfigForm.View()
	if strings.Contains(view, "Django Ninja") {
		t.Errorf("the add-on list was built before the Django version was known, "+
			"so it still offers ninja on 6.1:\n%s", view)
	}
}
