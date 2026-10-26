package scaffold

import (
	"context"
	"fmt"

	"github.com/vaibdix/django-stack/internal/config"
)

func (e *Env) applyHTMX(ctx context.Context) error {
	se, err := e.settings()
	if err != nil {
		return err
	}

	if err := se.AddToList(installedApps, appHTMX); err != nil {
		return fmt.Errorf("add %s to %s: %w", appHTMX, installedApps, err)
	}

	if err := se.AddToList(middlewareList, middlewareHTMX); err != nil {
		return fmt.Errorf("add %s to %s: %w", middlewareHTMX, middlewareList, err)
	}

	if err := se.Save(); err != nil {
		return err
	}
	e.Step(fmt.Sprintf("Added %s to %s and %s to %s.", appHTMX, installedApps, "HtmxMiddleware", middlewareList))

	return nil
}

func htmxAddOn() addOn {
	return addOn{
		enabled: config.Choices.UsesHTMX,
		steps:   []phasedStep{{phaseFeatures, step{name: "setup htmx", label: "Setting up HTMX...", run: (*Env).applyHTMX}}},
		contribute: func(*Env) contribution {
			return contribution{runtime: []string{pkgHTMX}}
		},
	}
}
