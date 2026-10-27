package render

import (
	"github.com/vaibdix/django-stack/internal/config"
)

type VerificationCheck struct {
	Name string
	Args []string
}

func (p Project) VerificationChecks() []VerificationCheck {
	checks := []VerificationCheck{
		{Name: "system check", Args: []string{ManageFile, "check"}},
		{Name: "test suite", Args: append([]string{ManageFile, "test", "--noinput"}, p.testSettings()...)},
	}
	for _, c := range p.contributions() {
		checks = append(checks, c.checks...)
	}
	return checks
}

func (p Project) testSettings() []string {
	if p.Cfg.Has(config.MultiEnv) {
		return []string{"--settings=" + p.Cfg.ModuleName() + ".settings.test"}
	}
	return nil
}
