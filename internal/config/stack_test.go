package config

import (
	"strings"
	"testing"
)

func TestEveryStackMapsToTheFlagsTheEngineGenerates(t *testing.T) {
	t.Parallel()

	cases := map[Stack]struct{ templates, tailwind, htmx bool }{
		StackStandard:     {templates: true},
		StackHTMX:         {templates: true, htmx: true},
		StackTailwind:     {templates: true, tailwind: true},
		StackTailwindHTMX: {templates: true, tailwind: true, htmx: true},
		StackAPIOnly:      {},
	}

	for stack, want := range cases {
		var p Project
		p.ApplyFeatures(stack.Features())

		if p.CreateTemplates != want.templates {
			t.Errorf("%s: CreateTemplates = %v, want %v", stack, p.CreateTemplates, want.templates)
		}
		if p.SetupTailwind != want.tailwind {
			t.Errorf("%s: SetupTailwind = %v, want %v", stack, p.SetupTailwind, want.tailwind)
		}
		if p.SetupHTMX != want.htmx {
			t.Errorf("%s: SetupHTMX = %v, want %v", stack, p.SetupHTMX, want.htmx)
		}
		if p.APIOnly() == want.templates {
			t.Errorf("%s: APIOnly() = %v alongside CreateTemplates = %v", stack, p.APIOnly(), p.CreateTemplates)
		}
	}
}

func TestTemplatesAndAppTemplatesAreNeverSplit(t *testing.T) {
	t.Parallel()

	// The reported bug: app CRUD templates open with {% extends 'base.html' %},
	// so generating them without the global templates yields TemplateDoesNotExist.
	for _, stack := range AllStacks() {
		var p Project
		p.ApplyFeatures(stack.Features())

		if p.CreateAppTemplates && !p.CreateTemplates {
			t.Errorf("%s generates app templates but no base.html for them to extend", stack)
		}
	}
}

func TestEveryTemplateVariantIsReachableFromSomeStack(t *testing.T) {
	t.Parallel()

	// One stack per testdata/golden/app_templates directory. If a variant stops
	// being reachable its goldens quietly become dead weight.
	seen := map[string]bool{}
	for _, stack := range AllStacks() {
		var p Project
		p.ApplyFeatures(stack.Features())
		if !p.CreateTemplates {
			continue
		}

		variant := "plain"
		switch {
		case p.SetupTailwind && p.SetupHTMX:
			variant = "tailwind_htmx"
		case p.SetupTailwind:
			variant = "tailwind"
		case p.SetupHTMX:
			variant = "htmx"
		}
		seen[variant] = true
	}

	for _, variant := range []string{"plain", "htmx", "tailwind", "tailwind_htmx"} {
		if !seen[variant] {
			t.Errorf("no stack produces the %q template variant, so no user can choose it", variant)
		}
	}
}

func TestAddOnsNeverCollideWithTheStackFeatures(t *testing.T) {
	t.Parallel()

	// The stack and the add-on list are applied together. If they overlapped, the
	// add-on multi-select would be able to turn a stack feature off behind the
	// user's back.
	stackFeatures := map[Feature]bool{}
	for _, stack := range AllStacks() {
		for _, f := range stack.Features() {
			stackFeatures[f] = true
		}
	}

	addOns := []Feature{
		FeatureGit, FeatureRestFramework, FeatureAllAuth,
		FeatureRuff, FeatureMypy, FeaturePreCommit, FeatureMultiEnv,
	}
	for _, f := range addOns {
		if stackFeatures[f] {
			t.Errorf("%q is offered as both a stack feature and an add-on", f)
		}
	}
}

func TestEveryStackIsLabelledAndDescribed(t *testing.T) {
	t.Parallel()

	labels := map[string]bool{}
	for _, stack := range AllStacks() {
		label, description := stack.Label(), stack.Description()
		if strings.TrimSpace(label) == "" || strings.TrimSpace(description) == "" {
			t.Errorf("%s has an empty label or description", stack)
		}
		if labels[label] {
			t.Errorf("two stacks share the label %q, so the picker cannot distinguish them", label)
		}
		labels[label] = true
	}
}
