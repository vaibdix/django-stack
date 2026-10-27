package scaffold

import (
	"context"

	"github.com/vaibdix/django-stack/internal/render"
)

func (e *Env) writeGitignore(ctx context.Context) error {
	if err := e.emitGroup(ctx, render.GroupGitignore); err != nil {
		return err
	}
	e.Step("Created: " + render.GitignoreFile)
	return nil
}
