package render

import (
	"slices"

	"github.com/vaibdix/django-stack/internal/config"
)

type urlContribution struct {
	home     string
	imports  []string
	blocks   []block
	routes   []route
	auth     bool
	appRoute bool
}

type settingsContribution struct {
	imports    []string
	apps       []entry
	middleware []entry
	blocks     []block
}

type contribution struct {
	runtime  []string
	dev      []string
	readme   func(*readmeData)
	urls     urlContribution
	checks   []VerificationCheck
	settings settingsContribution
}

type feature struct {
	enabled    func(config.Spec) bool
	contribute func(p Project) contribution
}

func features() []feature {
	return []feature{
		pydanticFeature(),
		tailwindFeature(),
		htmxFeature(),
		allAuthFeature(),
		structlogFeature(),
		sentryFeature(),
		cspFeature(),
		ruffFeature(),
		restFrameworkFeature(),
		ninjaFeature(),
		mypyFeature(),
		multiEnvFeature(),
		preCommitFeature(),
	}
}

func (p Project) contributions() []contribution {
	var all []contribution
	for _, f := range features() {
		if f.enabled(p.Cfg) {
			all = append(all, f.contribute(p))
		}
	}
	return all
}

func (p Project) contributed(field func(contribution) []string) []string {
	var all []string
	for _, c := range p.contributions() {
		all = append(all, field(c)...)
	}
	return all
}

func contributedTo[T any](p Project, field func(settingsContribution) []T) []T {
	var all []T
	for _, c := range p.contributions() {
		all = append(all, field(c.settings)...)
	}
	return all
}

func (p Project) urlContributions() []urlContribution {
	var all []urlContribution
	for _, c := range p.contributions() {
		all = append(all, c.urls)
	}
	return all
}

func anyURL(urls []urlContribution, flag func(urlContribution) bool) bool {
	return slices.ContainsFunc(urls, flag)
}
