package tui

import (
	"strings"

	"charm.land/huh/v2"

	"github.com/vaibdix/django-stack/internal/config"
)

func buildStackForm(target *config.Stack) *huh.Form {
	stacks := config.AllStacks()
	options := make([]huh.Option[config.Stack], 0, len(stacks))
	for _, stack := range stacks {
		options = append(options, huh.NewOption(stack.Spec().Label, stack))
	}

	return selectForm("Choose your stack", "This picks the HTML templates the project is built from.", options, target, nil)
}

func joinAddOns(addOns []config.AddOn, sep string) string {
	if len(addOns) == 0 {
		return "none"
	}
	names := make([]string, len(addOns))
	for i, addOn := range addOns {
		names[i] = string(addOn)
	}
	return strings.Join(names, sep)
}

func addOnSummary(selected, added []config.AddOn) []string {
	lines := []string{"Add-ons: " + joinAddOns(selected, ", ")}
	for _, addOn := range added {
		lines = append(lines, "Added "+string(addOn)+", which "+joinAddOns(config.RequiredBy(addOn, selected), " and ")+" needs")
	}
	return lines
}
