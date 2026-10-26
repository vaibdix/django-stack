package scaffold

import (
	"slices"

	"github.com/vaibdix/django-stack/internal/config"
)

type phase int

const (
	phaseVCS phase = iota
	phaseFeatures
	phaseAPI
	phaseAssets
	phaseLayout
	phaseHooks
)

type phasedStep struct {
	phase phase
	step  step
}

type urlContribution struct {
	home     string
	imports  []string
	blocks   []string
	patterns []string
	auth     bool
	appRoute bool
}

type contribution struct {
	runtime []string
	dev     []string
	readme  func(*readmeData)
	urls    urlContribution
	checks  []verificationCheck
}

type addOn struct {
	enabled    func(config.Choices) bool
	steps      []phasedStep
	contribute func(e *Env) contribution
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
				steps = append(steps, s.step)
			}
		}
	}
	return steps
}

func (e *Env) contributions() []contribution {
	var all []contribution
	for _, a := range addOns() {
		if a.contribute != nil && a.enabled(e.cfg) {
			all = append(all, a.contribute(e))
		}
	}
	return all
}

func (e *Env) contributed(field func(contribution) []string) []string {
	var all []string
	for _, c := range e.contributions() {
		all = append(all, field(c)...)
	}
	return all
}

func (e *Env) urlContributions() []urlContribution {
	var all []urlContribution
	for _, c := range e.contributions() {
		all = append(all, c.urls)
	}
	return all
}

func anyURL(urls []urlContribution, flag func(urlContribution) bool) bool {
	return slices.ContainsFunc(urls, flag)
}
