package scaffold

import (
	"context"
	"fmt"

	"github.com/vaibdix/django-stack/internal/config"
)

func gitAddOn() addOn {
	return addOn{
		enabled: has(config.Git),
		steps:   []phasedStep{{phase: phaseVCS, step: step{name: "initialize git", label: "Initializing the git repository...", run: (*Env).initGitRepository}}},
	}
}

func (e *Env) initGitRepository(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, defaultCommandTimeout)
	defer cancel()

	if _, err := e.runIn(ctx, e.dir, "git", "init"); err != nil {
		return fmt.Errorf("initialize Git repository: %w", err)
	}
	e.Step("Git repository initialized.")
	return nil
}
