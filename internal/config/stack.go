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

func (p Project) APIOnly() bool {
	return !p.CreateTemplates
}
