package scaffold

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

type PyprojectEditor struct {
	path    string
	content string
}

// NewPyprojectEditor reads pyproject.toml, or starts one when the project has
// none yet. Any other read failure is returned: treating it as "no file"
// would overwrite whatever is there on Save.
func NewPyprojectEditor(projectPath, projectName string, metadata PythonToolingMetadata) (*PyprojectEditor, error) {
	path := filepath.Join(projectPath, PyprojectFile)

	data, err := os.ReadFile(path)
	switch {
	case err == nil:
		return &PyprojectEditor{path: path, content: string(data)}, nil
	case !errors.Is(err, fs.ErrNotExist):
		return nil, fmt.Errorf("failed to read %s: %w", PyprojectFile, err)
	}

	moduleName := strings.ReplaceAll(projectName, "-", "_")
	content := fmt.Sprintf(`[project]
name = "%s"
version = "0.1.0"
requires-python = "%s"
`, moduleName, metadata.RequiresPython)

	return &PyprojectEditor{path: path, content: content}, nil
}

func (p *PyprojectEditor) AppendSection(section string) {
	p.content += "\n" + section
}

// SetProjectDependencies writes [project].dependencies, replacing an existing
// array or adding one after requires-python.
func (p *PyprojectEditor) SetProjectDependencies(dependencies []string) error {
	block := renderTomlArray("dependencies", dependencies)

	if start := strings.Index(p.content, "\ndependencies = ["); start != -1 {
		end := strings.Index(p.content[start:], "\n]\n")
		if end == -1 {
			return &AnchorError{Edit: "the project dependencies", Anchor: "the closing ] of dependencies"}
		}
		p.content = p.content[:start+1] + block + p.content[start+end+len("\n]\n"):]
		return nil
	}

	const anchor = "requires-python = "
	idx := strings.Index(p.content, anchor)
	if idx == -1 {
		return &AnchorError{Edit: "the project dependencies", Anchor: anchor}
	}
	insertAt := len(p.content)
	if lineEnd := strings.Index(p.content[idx:], "\n"); lineEnd != -1 {
		insertAt = idx + lineEnd + 1
	}
	p.content = p.content[:insertAt] + block + p.content[insertAt:]
	return nil
}

// SetDependencyGroup adds a [dependency-groups] table holding one group. The
// scaffold writes the table once; finding one already there means a step ran
// twice or the file was edited, and silently keeping the old one would drop
// the group being set.
func (p *PyprojectEditor) SetDependencyGroup(name string, dependencies []string) error {
	if p.Contains("[dependency-groups]") {
		return fmt.Errorf("%s already has a [dependency-groups] table", PyprojectFile)
	}
	p.AppendSection("[dependency-groups]\n" + renderTomlArray(name, dependencies))
	return nil
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
