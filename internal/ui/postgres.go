package ui

import (
	"fmt"
	"os"
	"path/filepath"

	"django-stack/internal/templates"
)

func (m *Model) CreatePostgresReadme(projectPath string) error {
	readmeContent := templates.PostgresReadmeTemplate()

	readmePath := filepath.Join(projectPath, "README-postgres.md")
	if err := os.WriteFile(readmePath, []byte(readmeContent), 0644); err != nil {
		return fmt.Errorf("failed to create README-postgres.md: %w", err)
	}

	m.AppendStepMessage("Created: README-postgres.md")
	return nil
}
