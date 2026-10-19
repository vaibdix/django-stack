package pyenv

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/vaibdix/django-cli/internal/scaffold/command"
)

type uvPythonEntry struct {
	Key            string `json:"key"`
	Version        string `json:"version"`
	Path           string `json:"path"`
	Implementation string `json:"implementation"`
	Variant        string `json:"variant"`
}

func (e uvPythonEntry) usable() bool {
	if e.Path == "" {
		return false
	}
	if e.Implementation != "" && !strings.EqualFold(e.Implementation, "cpython") {
		return false
	}
	return e.Variant == "" || strings.EqualFold(e.Variant, "default")
}

// uvManagedPythonPaths lists the interpreters uv has installed. A uv that is
// not installed lists none; a uv that is installed but cannot list them is an
// error, reported rather than answered by scraping its human-readable output.
func uvManagedPythonPaths(ctx context.Context, run command.Runner) ([]string, error) {
	if !CommandAvailable("uv") {
		return nil, nil
	}

	output, err := probe(ctx, run, "uv", "python", "list", "--only-installed", "--output-format", "json")
	if err != nil {
		return nil, fmt.Errorf("uv python list failed: %w", err)
	}

	var entries []uvPythonEntry
	if err := json.Unmarshal(output, &entries); err != nil {
		return nil, fmt.Errorf("uv python list printed JSON this version of django-stack cannot read: %w", err)
	}

	paths := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.usable() {
			paths = append(paths, entry.Path)
		}
	}
	return paths, nil
}
