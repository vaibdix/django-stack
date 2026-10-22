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

	return selectForm("Choose your stack", "This picks the HTML templates the project is built from.", options, target, nil)
}

func (m *Model) selectedFeatures() []config.Feature {
	return append(m.selectedStack.Features(), m.selectedOptions...)
}
