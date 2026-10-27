package scaffold

import (
	"context"

	"github.com/vaibdix/django-stack/internal/render"
)

func (e *Env) applyAllAuth(ctx context.Context) error {
	if e.Cfg.CreateTemplates() {
		if err := e.emitGroup(render.GroupAllAuth); err != nil {
			return err
		}

		e.Step("Created styled AllAuth templates (login, signup, logout).")
	}

	return nil
}
