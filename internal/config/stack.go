package config

import (
	"slices"

	"github.com/vaibdix/django-stack/internal/catalog"
)

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

var stacks = stacksFrom(catalog.Load())

func stacksFrom(c catalog.Catalog) []StackSpec {
	specs := make([]StackSpec, len(c.Stacks))
	for i, row := range c.Stacks {
		specs[i] = StackSpec{Name: Stack(row.Name), Label: row.Label, Templates: row.Templates, Tailwind: row.Tailwind, HTMX: row.HTMX}
	}
	return specs
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

func (p Choices) APIOnly() bool {
	return !p.CreateTemplates()
}
