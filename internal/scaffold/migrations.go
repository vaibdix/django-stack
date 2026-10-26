package scaffold

import (
	"context"
	"fmt"

	"github.com/vaibdix/django-stack/internal/pyenv"
)

func (e *Env) writeGitignore(ctx context.Context) error {
	if err := e.write(gitignoreFile, []byte(renderStatic("gitignore.tmpl")), fileMode); err != nil {
		return err
	}
	e.Step("Created: " + gitignoreFile)
	return nil
}

func (e *Env) runMigrations(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, defaultCommandTimeout)
	defer cancel()

	pythonPath := pyenv.InterpreterPath(e.dir)

	if _, err := e.runIn(ctx, e.dir, pythonPath, manageFile, "makemigrations"); err != nil {
		return fmt.Errorf("create migrations: %w", err)
	}
	e.Step("Created database migrations.")

	if _, err := e.runIn(ctx, e.dir, pythonPath, manageFile, "migrate"); err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}
	e.Step("Applied database migrations.")
	e.Step("Database configured via .env file.")

	if e.cfg.App != "" {
		if _, err := e.runIn(ctx, e.dir, pythonPath, manageFile, "create_sample_data"); err != nil {
			return fmt.Errorf("create sample data: %w", err)
		}
		e.Step("Created sample data for the example app.")
	}

	return nil
}
