package tui

import (
	"github.com/charmbracelet/huh"

	"github.com/vaibdix/django-stack/internal/config"
)

func buildStackForm(target *config.Stack) *huh.Form {
	stacks := config.AllStacks()
	options := make([]huh.Option[config.Stack], 0, len(stacks))
	for _, stack := range stacks {
		options = append(options, huh.NewOption(stack.Label(), stack))
	}

	return newForm(
		huh.NewGroup(
			huh.NewSelect[config.Stack]().
				Title("Choose your stack").
				Description("This picks the HTML templates the project is built from.").
				Options(options...).
				Value(target),
		),
	)
}

func (m *Model) selectedFeatures() []config.Feature {
	return append(m.selectedStack.Features(), m.selectedOptions...)
}
