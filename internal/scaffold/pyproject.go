package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type PyprojectEditor struct {
	path    string
	content string
}

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

func (p *PyprojectEditor) AppendSection(section string) {
	p.content += "\n" + section
}

func (p *PyprojectEditor) Contains(substr string) bool {
	return strings.Contains(p.content, substr)
}

func (p *PyprojectEditor) Save() error {
	if err := os.WriteFile(p.path, []byte(p.content), 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", PyprojectFile, err)
	}
	return nil
}
