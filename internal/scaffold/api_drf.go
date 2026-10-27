package scaffold

import (
	"context"

	"github.com/vaibdix/django-stack/internal/render"
)

func (e *Env) writeAPIPackage(ctx context.Context) error {
	if err := e.emitGroup(ctx, render.GroupAPI); err != nil {
		return err
	}
	e.Step("Created api/ package.")
	return nil
}
