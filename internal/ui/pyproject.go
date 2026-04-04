package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// PyprojectEditor manages creation and appending to a pyproject.toml file.
type PyprojectEditor struct {
	path    string
	content string
}

// NewPyprojectEditor creates or reads an existing pyproject.toml in projectPath.
func NewPyprojectEditor(projectPath, projectName string, metadata PythonToolingMetadata) *PyprojectEditor {
	path := filepath.Join(projectPath, PyprojectFile)

	data, err := os.ReadFile(path)
	if err == nil {
		return &PyprojectEditor{path: path, content: string(data)}
	}

	moduleName := strings.ReplaceAll(projectName, "-", "_")
	content := fmt.Sprintf(`[project]
name = "%s"
version = "0.1.0"
requires-python = "%s"
`, moduleName, metadata.RequiresPython)

	return &PyprojectEditor{path: path, content: content}
}

// AppendSection appends a TOML section block to the file content.
func (p *PyprojectEditor) AppendSection(section string) {
	p.content += "\n" + section
}

// Contains checks if the content includes the given substring.
func (p *PyprojectEditor) Contains(substr string) bool {
	return strings.Contains(p.content, substr)
}

// Save writes the content to disk.
func (p *PyprojectEditor) Save() error {
	if err := os.WriteFile(p.path, []byte(p.content), 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", PyprojectFile, err)
	}
	return nil
}
