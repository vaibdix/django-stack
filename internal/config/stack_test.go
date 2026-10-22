package config

import (
	"strings"
	"testing"
	"time"
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

func TestEveryTemplateVariantIsReachableFromSomeStack(t *testing.T) {
	t.Parallel()

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

func TestEveryStackHasItsOwnLabel(t *testing.T) {
	t.Parallel()

	labels := map[string]bool{}
	for _, stack := range AllStacks() {
		label := stack.Label()
		if strings.TrimSpace(label) == "" {
			t.Errorf("%s has an empty label", stack)
		}
		if labels[label] {
			t.Errorf("two stacks share the label %q, so the picker cannot distinguish them", label)
		}
		labels[label] = true
	}
}

func TestASeriesDropsOutWhenItsSecuritySupportEnds(t *testing.T) {
	t.Parallel()

	today := time.Date(2026, time.October, 16, 0, 0, 0, 0, time.UTC)
	series := map[string]bool{}
	for _, rule := range SupportedDjangoSeries(today) {
		series[rule.Series] = true
	}
	for _, want := range []string{"5.2", "6.0", "6.1"} {
		if !series[want] {
			t.Errorf("Django %s is still supported on %s but was dropped", want, today.Format(time.DateOnly))
		}
	}
	for _, eol := range []string{"4.2", "5.0", "5.1"} {
		if series[eol] {
			t.Errorf("Django %s is past end of life but is still offered", eol)
		}
	}

	later := time.Date(2027, time.May, 1, 0, 0, 0, 0, time.UTC)
	for _, rule := range SupportedDjangoSeries(later) {
		if rule.Series == "6.0" {
			t.Error("Django 6.0 is still offered after its April 2027 end of support")
		}
	}
}

func TestDjangoSeriesIsTheMajorAndMinorVersion(t *testing.T) {
	t.Parallel()

	for version, want := range map[string]string{"6.1": "6.1", "6.1.2": "6.1", "6.0.9": "6.0", "5.2.18": "5.2"} {
		if got := DjangoSeries(version); got != want {
			t.Errorf("DjangoSeries(%q) = %q, want %q", version, got, want)
		}
	}
}
