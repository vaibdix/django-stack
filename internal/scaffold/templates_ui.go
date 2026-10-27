package scaffold

import (
	"context"

	"github.com/vaibdix/django-stack/internal/render"
)

func (e *Env) applyGlobalTemplates(ctx context.Context) error {
	if err := e.emitGroup(ctx, render.GroupTemplates); err != nil {
		return err
	}
	e.Step("Created global templates and static files.")
	return nil
}

func (e *Env) writeProjectURLs(ctx context.Context) error {
	return e.emitGroup(ctx, render.GroupURLs)
}
