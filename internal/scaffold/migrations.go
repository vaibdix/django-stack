package scaffold

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/vaibdix/django-stack/internal/pyenv"
)

func (e *Env) initGitRepository(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, defaultCommandTimeout)
	defer cancel()

	if _, err := e.runIn(ctx, e.dir, "git", "init"); err != nil {
		return fmt.Errorf("failed to initialize Git repository: %w", err)
	}
	e.Step("Git repository initialized.")
	return nil
}

func (e *Env) writeGitignore(ctx context.Context) error {
	if err := os.WriteFile(filepath.Join(e.dir, gitignoreFile), []byte(renderStatic("gitignore.tmpl")), 0644); err != nil {
		return fmt.Errorf("failed to create %s: %w", gitignoreFile, err)
	}
	e.Step("Created: " + gitignoreFile)
	return nil
}

func (e *Env) runMigrations(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, defaultCommandTimeout)
	defer cancel()

	pythonPath := pyenv.InterpreterPath(e.dir)

	if _, err := e.runIn(ctx, e.dir, pythonPath, manageFile, "makemigrations"); err != nil {
		return fmt.Errorf("failed to create migrations: %w", err)
	}
	e.Step("Created database migrations.")

	if _, err := e.runIn(ctx, e.dir, pythonPath, manageFile, "migrate"); err != nil {
		return fmt.Errorf("failed to apply migrations: %w", err)
	}
	e.Step("Applied database migrations.")
	e.Step("Database configured via .env file.")

	if e.cfg.App != "" {
		if _, err := e.runIn(ctx, e.dir, pythonPath, manageFile, "create_sample_data"); err != nil {
			return fmt.Errorf("failed to create sample data: %w", err)
		}
		e.Step("Created sample data for the example app.")
	}

	return nil
}
