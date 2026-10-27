package scaffold

import (
	"context"
	"fmt"

	"github.com/vaibdix/django-stack/internal/pyenv"
	"github.com/vaibdix/django-stack/internal/render"
)

func (e *Env) runMigrations(ctx context.Context) error {
	pythonPath := pyenv.InterpreterPath(e.dir)

	if _, err := e.runIn(ctx, e.dir, pythonPath, render.ManageFile, "migrate"); err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}
	e.Step("Applied database migrations.")
	e.Step("Database configured via .env file.")

	if e.Cfg.App().Ok() {
		if _, err := e.runIn(ctx, e.dir, pythonPath, render.ManageFile, "create_sample_data"); err != nil {
			return fmt.Errorf("create sample data: %w", err)
		}
		e.Step("Created sample data for the example app.")
	}

	return nil
}
