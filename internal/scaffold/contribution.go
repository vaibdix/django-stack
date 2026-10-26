package scaffold

import (
	"slices"

	"github.com/vaibdix/django-stack/internal/config"
)

type urlContribution struct {
	home     string
	imports  []string
	blocks   []string
	patterns []string
	auth     bool
	appRoute bool
}

type settingsContribution struct {
	imports    []string
	apps       []string
	lateApps   []string
	middleware []string
	blocks     []string
	lateBlocks []string
}

type contribution struct {
	runtime  []string
	dev      []string
	readme   func(*readmeData)
	urls     urlContribution
	checks   []verificationCheck
	settings settingsContribution
}

type feature struct {
	enabled    func(config.Choices) bool
	contribute func(p project) contribution
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

func (p project) contributions() []contribution {
	var all []contribution
	for _, f := range features() {
		if f.enabled(p.cfg) {
			all = append(all, f.contribute(p))
		}
	}
	return all
}

func (p project) contributed(field func(contribution) []string) []string {
	var all []string
	for _, c := range p.contributions() {
		all = append(all, field(c)...)
	}
	return all
}

func (p project) contributedSettings(field func(settingsContribution) []string) []string {
	var all []string
	for _, c := range p.contributions() {
		all = append(all, field(c.settings)...)
	}
	return all
}

func (p project) urlContributions() []urlContribution {
	var all []urlContribution
	for _, c := range p.contributions() {
		all = append(all, c.urls)
	}
	return all
}

func anyURL(urls []urlContribution, flag func(urlContribution) bool) bool {
	return slices.ContainsFunc(urls, flag)
}
