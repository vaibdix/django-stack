package scaffold

import (
	"context"

	"github.com/vaibdix/django-stack/internal/render"
)

func (e *Env) createAccountsApp(ctx context.Context) error {
	if err := e.emitGroup(ctx, render.GroupSharedApps); err != nil {
		return err
	}

	e.Step("Created the core and accounts apps; the custom User model lands before the first migration.")
	return nil
}
