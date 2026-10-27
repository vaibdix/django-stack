package tui

import (
	"fmt"
	"strings"

	"charm.land/huh/v2"

	"github.com/vaibdix/django-stack/internal/config"
	"github.com/vaibdix/django-stack/internal/scaffold"
)

func buildStackForm(target *config.Stack, host scaffold.Host) *huh.Form {
	stacks := config.AllStacks()
	options := make([]huh.Option[config.Stack], 0, len(stacks))
	for _, stack := range stacks {
		options = append(options, huh.NewOption(optionLabel(stack.Spec().Label, host.Missing(config.StackNeeds(stack))), stack))
	}

	return selectForm("Choose your stack", "This picks the HTML templates the project is built from.", options, target, func(stack config.Stack) error {
		if missing := host.Missing(config.StackNeeds(stack)); missing != nil {
			return fmt.Errorf("%s is not available on this machine: %w", stack.Spec().Label, missing)
		}
		return nil
	})
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

func addOnSummary(selected []config.AddOn) []string {
	lines := []string{"Add-ons: " + joinAddOns(selected, ", ")}
	_, adjustments, err := config.NewFeatureSet(selected)
	if err != nil {
		return lines
	}
	for _, adjustment := range adjustments {
		lines = append(lines, adjustment.String())
	}
	return lines
}
