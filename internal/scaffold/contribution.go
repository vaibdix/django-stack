package scaffold

import (
	"slices"

	"github.com/vaibdix/django-stack/internal/config"
)

type phase int

const (
	phaseInstall phase = iota
	phaseVCS
	phaseFeatures
	phaseAPI
	phaseAssets
	phaseLayout
	phaseHooks
	phaseMigrations
)

type phasedStep struct {
	phase     phase
	step      step
	alongside bool
}

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

type addOn struct {
	enabled    func(config.Choices) bool
	steps      []phasedStep
	contribute func(p project) contribution
}

func addOns() []addOn {
	return []addOn{
		pydanticAddOn(),
		gitAddOn(),
		tailwindAddOn(),
		htmxAddOn(),
		allAuthAddOn(),
		structlogAddOn(),
		sentryAddOn(),
		cspAddOn(),
		ruffAddOn(),
		restFrameworkAddOn(),
		ninjaAddOn(),
		mypyAddOn(),
		multiEnvAddOn(),
		preCommitAddOn(),
	}
}

func addOnSteps(p phase) []step {
	var steps []step
	for _, a := range addOns() {
		for _, s := range a.steps {
			if s.phase == p {
				s.step.when = a.enabled
				s.step.alongside = s.alongside
				steps = append(steps, s.step)
			}
		}
	}
	return steps
}

func (p project) contributions() []contribution {
	var all []contribution
	for _, a := range addOns() {
		if a.contribute != nil && a.enabled(p.cfg) {
			all = append(all, a.contribute(p))
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
