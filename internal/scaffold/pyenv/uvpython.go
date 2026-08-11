package pyenv

import (
	"encoding/json"
	"strings"
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

func uvManagedPythonPaths() []string {
	if !CommandAvailable("uv") {
		return nil
	}

	output, err := probe("uv", "python", "list", "--only-installed", "--output-format", "json")
	if err != nil {
		return uvManagedPythonPathsFromText()
	}

	var entries []uvPythonEntry
	if err := json.Unmarshal(output, &entries); err != nil {
		return uvManagedPythonPathsFromText()
	}

	paths := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.usable() {
			paths = append(paths, entry.Path)
		}
	}
	return paths
}

func uvManagedPythonPathsFromText() []string {
	output, err := probe("uv", "python", "list", "--only-installed")
	if err != nil {
		return nil
	}
	return parseUVTextListing(string(output))
}

func parseUVTextListing(listing string) []string {
	var paths []string
	for line := range strings.SplitSeq(listing, "\n") {
		key, rest, ok := strings.Cut(strings.TrimSpace(line), " ")
		if !ok || !strings.HasPrefix(key, "cpython-") || strings.Contains(key, "+") {
			continue
		}

		path, _, _ := strings.Cut(strings.TrimSpace(rest), " ->")
		path = strings.TrimSpace(path)
		if path != "" && !strings.HasPrefix(path, "<") {
			paths = append(paths, path)
		}
	}
	return paths
}
