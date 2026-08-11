package tui

import (
	"github.com/charmbracelet/huh"

	"github.com/vaibdix/django-cli/internal/config"
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

// applySelectedStack folds the stack back into SelectedOptions so ApplyFeatures
// stays the single place that turns a selection into flags, and so returning to
// this step with the left arrow can recover the choice from the model.
func (m *Model) applySelectedStack() {
	kept := make([]config.Feature, 0, len(m.SelectedOptions))
	for _, feature := range m.SelectedOptions {
		if !isStackFeature(feature) {
			kept = append(kept, feature)
		}
	}

	m.SelectedOptions = append(m.SelectedStack.Features(), kept...)
}

func isStackFeature(feature config.Feature) bool {
	switch feature {
	case config.FeatureStandardProject, config.FeatureTailwind, config.FeatureHTMX:
		return true
	default:
		return false
	}
}
