package config

import "slices"

type Stack string

const (
	StackStandard     Stack = "standard"
	StackHTMX         Stack = "htmx"
	StackTailwind     Stack = "tailwind"
	StackTailwindHTMX Stack = "tailwind-htmx"
	StackAPIOnly      Stack = "api-only"
)

type StackSpec struct {
	Name      Stack
	Label     string
	Templates bool
	Tailwind  bool
	HTMX      bool
}

var stacks = []StackSpec{
	{Name: StackStandard, Label: "Standard Django (templates & static files)", Templates: true},
	{Name: StackHTMX, Label: "Standard + HTMX", Templates: true, HTMX: true},
	{Name: StackTailwind, Label: "Standard + Tailwind CSS v4", Templates: true, Tailwind: true},
	{Name: StackTailwindHTMX, Label: "Standard + Tailwind CSS v4 + HTMX", Templates: true, Tailwind: true, HTMX: true},
	{Name: StackAPIOnly, Label: "API only (no templates)"},
}

func AllStacks() []Stack {
	names := make([]Stack, len(stacks))
	for i, spec := range stacks {
		names[i] = spec.Name
	}
	return names
}

func (s Stack) Valid() bool {
	return slices.ContainsFunc(stacks, func(spec StackSpec) bool { return spec.Name == s })
}

func (s Stack) Spec() StackSpec {
	i := slices.IndexFunc(stacks, func(spec StackSpec) bool { return spec.Name == s })
	if i < 0 {
		return StackSpec{Name: s, Label: string(s)}
	}
	return stacks[i]
}

func (p Project) APIOnly() bool {
	return !p.CreateTemplates()
}
