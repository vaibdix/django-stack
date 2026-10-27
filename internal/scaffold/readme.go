package scaffold

import (
	"context"

	"github.com/vaibdix/django-stack/internal/render"
)

func (e *Env) writeReadme(ctx context.Context) error {
	if err := e.emitGroup(ctx, render.GroupReadme); err != nil {
		return err
	}

	e.Step("Created: README.md (setup, tests, linting and deployment)")
	return nil
}
