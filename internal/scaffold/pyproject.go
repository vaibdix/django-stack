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

func (p *PyprojectEditor) SetProjectDependencies(dependencies []string) {
	block := renderTomlArray("dependencies", dependencies)

	if start := strings.Index(p.content, "\ndependencies = ["); start != -1 {
		if end := strings.Index(p.content[start:], "\n]\n"); end != -1 {
			p.content = p.content[:start+1] + block + p.content[start+end+len("\n]\n"):]
			return
		}
	}

	const anchor = "requires-python = "
	idx := strings.Index(p.content, anchor)
	if idx == -1 {
		p.content = "[project]\n" + block + "\n" + p.content
		return
	}

	lineEnd := strings.Index(p.content[idx:], "\n")
	if lineEnd == -1 {
		p.content += "\n" + block
		return
	}

	insertAt := idx + lineEnd + 1
	p.content = p.content[:insertAt] + block + p.content[insertAt:]
}

func (p *PyprojectEditor) SetDependencyGroup(name string, dependencies []string) {
	if p.Contains("[dependency-groups]") {
		return
	}
	p.AppendSection("[dependency-groups]\n" + renderTomlArray(name, dependencies))
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
