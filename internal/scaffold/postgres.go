package scaffold

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/vaibdix/django-cli/internal/assets"
)

func (e *Env) writePostgresReadme(projectPath string) error {
	readmeContent := assets.PostgresReadme()

	readmePath := filepath.Join(projectPath, "README-postgres.md")
	if err := os.WriteFile(readmePath, []byte(readmeContent), 0644); err != nil {
		return fmt.Errorf("failed to create README-postgres.md: %w", err)
	}

	e.Step("Created: README-postgres.md")
	return nil
}
