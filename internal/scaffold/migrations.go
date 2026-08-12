package scaffold

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/vaibdix/django-cli/internal/scaffold/pyenv"
)

func (e *Env) initGitRepository(projectPath string) error {
	if !e.Cfg.InitializeGit {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultCommandTimeout)
	defer cancel()

	if _, err := e.runIn(ctx, projectPath, "git", "init"); err != nil {
		return fmt.Errorf("failed to initialize Git repository: %w", err)
	}
	e.Step("Git repository initialized.")

	gitignoreContent := renderStatic("gitignore.tmpl")
	if err := os.WriteFile(filepath.Join(projectPath, ".gitignore"), []byte(gitignoreContent), 0644); err != nil {
		return fmt.Errorf("failed to create .gitignore: %w", err)
	}
	e.Step(".gitignore file created.")

	return nil
}

func (e *Env) runMigrations(projectPath string) error {

	ctx, cancel := context.WithTimeout(context.Background(), defaultCommandTimeout)
	defer cancel()

	pythonPath := pyenv.InterpreterPath(projectPath)

	if _, err := e.runIn(ctx, projectPath, pythonPath, ManageFile, "makemigrations"); err != nil {
		return fmt.Errorf("failed to create migrations: %w", err)
	}
	e.Step("Created database migrations.")

	if _, err := e.runIn(ctx, projectPath, pythonPath, ManageFile, "migrate"); err != nil {
		return fmt.Errorf("failed to apply migrations: %w", err)
	}
	e.Step("Applied database migrations.")
	e.Step("Database configured via .env file.")

	if e.Cfg.SetupMultiEnv {
		if _, err := e.runIn(ctx, projectPath, pythonPath, ManageFile, "createcachetable"); err != nil {
			return fmt.Errorf("failed to create the cache table production settings rely on: %w", err)
		}
		e.Step("Created the shared cache table, so API throttling counts across workers.")
	}

	if e.Cfg.HasAPI() && e.Cfg.AppName != "" {
		if _, err := e.runIn(ctx, projectPath, pythonPath, ManageFile, "create_sample_data"); err != nil {
			return fmt.Errorf("failed to create sample data: %w", err)
		}
		e.Step("Created sample data for the API.")
	}

	return nil
}
