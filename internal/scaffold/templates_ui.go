package scaffold

import (
	"context"

	"github.com/vaibdix/django-stack/internal/render"
)

func (e *Env) applyGlobalTemplates() error {
	if err := e.emitGroup(render.GroupTemplates); err != nil {
		return err
	}
	e.Step("Created global templates and static files.")
	return nil
}

func (e *Env) writeProjectURLs(ctx context.Context) error {
	return e.emitGroup(render.GroupURLs)
}
