package scaffold

import (
	"context"

	"github.com/vaibdix/django-stack/internal/render"
)

func (e *Env) writeVSCodeConfig(ctx context.Context) error {
	return e.emitGroup(ctx, render.GroupWorkspace)
}
