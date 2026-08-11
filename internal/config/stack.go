package config

type Stack string

const (
	StackStandard     Stack = "standard"
	StackHTMX         Stack = "htmx"
	StackTailwind     Stack = "tailwind"
	StackTailwindHTMX Stack = "tailwind-htmx"
	StackAPIOnly      Stack = "api-only"
)

func AllStacks() []Stack {
	return []Stack{StackStandard, StackHTMX, StackTailwind, StackTailwindHTMX, StackAPIOnly}
}

func (s Stack) Features() []Feature {
	switch s {
	case StackHTMX:
		return []Feature{FeatureStandardProject, FeatureHTMX}
	case StackTailwind:
		return []Feature{FeatureStandardProject, FeatureTailwind}
	case StackTailwindHTMX:
		return []Feature{FeatureStandardProject, FeatureTailwind, FeatureHTMX}
	case StackAPIOnly:
		return nil
	default:
		return []Feature{FeatureStandardProject}
	}
}

func (s Stack) Label() string {
	switch s {
	case StackHTMX:
		return "Standard + HTMX"
	case StackTailwind:
		return "Standard + Tailwind CSS v4"
	case StackTailwindHTMX:
		return "Standard + Tailwind CSS v4 + HTMX"
	case StackAPIOnly:
		return "API only (no templates)"
	default:
		return "Standard Django (templates & static files)"
	}
}

func (s Stack) Description() string {
	switch s {
	case StackHTMX:
		return "Server-rendered pages with HTMX for interactions, no build step."
	case StackTailwind:
		return "Server-rendered pages styled with Tailwind CSS v4."
	case StackTailwindHTMX:
		return "Tailwind CSS v4 for styling and HTMX for interactions."
	case StackAPIOnly:
		return "No HTML templates. Pair with the REST Framework add-on for a JSON API."
	default:
		return "Plain HTML templates and static files, no frontend toolchain."
	}
}

// StackFor recovers the stack a selection represents. Back-navigation rebuilds
// the form from the model, so the value has to survive a round trip through
// SelectedOptions rather than living only in the widget.
func StackFor(selected []Feature) Stack {
	var templates, tailwind, htmx bool
	for _, f := range selected {
		switch f {
		case FeatureStandardProject:
			templates = true
		case FeatureTailwind:
			tailwind = true
		case FeatureHTMX:
			htmx = true
		}
	}

	if !templates {
		return StackAPIOnly
	}
	switch {
	case tailwind && htmx:
		return StackTailwindHTMX
	case tailwind:
		return StackTailwind
	case htmx:
		return StackHTMX
	default:
		return StackStandard
	}
}

func (p Project) APIOnly() bool {
	return !p.CreateTemplates
}
