package scaffold

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

type pyprojectEditor struct {
	path    string
	content string
}

func newPyprojectEditor(projectPath, projectName string, metadata pythonToolingMetadata) (*pyprojectEditor, error) {
	path := filepath.Join(projectPath, pyprojectFile)

	data, err := os.ReadFile(path)
	switch {
	case err == nil:
		return &pyprojectEditor{path: path, content: string(data)}, nil
	case !errors.Is(err, fs.ErrNotExist):
		return nil, fmt.Errorf("failed to read %s: %w", pyprojectFile, err)
	}

	moduleName := strings.ReplaceAll(projectName, "-", "_")
	content := fmt.Sprintf(`[project]
name = "%s"
version = "0.1.0"
requires-python = "%s"
`, moduleName, metadata.RequiresPython)

	return &pyprojectEditor{path: path, content: content}, nil
}

func (p *pyprojectEditor) AppendSection(section string) {
	p.content += "\n" + section
}

func (p *pyprojectEditor) SetProjectDependencies(dependencies []string) error {
	block := renderTomlArray("dependencies", dependencies)

	if start := strings.Index(p.content, "\ndependencies = ["); start != -1 {
		end := strings.Index(p.content[start:], "\n]\n")
		if end == -1 {
			return &anchorError{Edit: "the project dependencies", Anchor: "the closing ] of dependencies"}
		}
		p.content = p.content[:start+1] + block + p.content[start+end+len("\n]\n"):]
		return nil
	}

	const anchor = "requires-python = "
	idx := strings.Index(p.content, anchor)
	if idx == -1 {
		return &anchorError{Edit: "the project dependencies", Anchor: anchor}
	}
	insertAt := len(p.content)
	if lineEnd := strings.Index(p.content[idx:], "\n"); lineEnd != -1 {
		insertAt = idx + lineEnd + 1
	}
	p.content = p.content[:insertAt] + block + p.content[insertAt:]
	return nil
}

func (p *pyprojectEditor) SetDependencyGroup(name string, dependencies []string) error {
	if p.Contains("[dependency-groups]") {
		return fmt.Errorf("%s already has a [dependency-groups] table", pyprojectFile)
	}
	p.AppendSection("[dependency-groups]\n" + renderTomlArray(name, dependencies))
	return nil
}

func (p *pyprojectEditor) Contains(substr string) bool {
	return strings.Contains(p.content, substr)
}

func (p *pyprojectEditor) Save() error {
	if err := os.WriteFile(p.path, []byte(p.content), 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", pyprojectFile, err)
	}
	return nil
}
