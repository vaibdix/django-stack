package render

import (
	"runtime"
	"sort"
	"strings"
)

const DevRequirementsFile = "requirements-dev.txt"

type PipReport struct {
	Version string `json:"version"`
	Install []struct {
		Metadata struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		} `json:"metadata"`
	} `json:"install"`
}

const PipReportVersion = "1"

func PinnedLines(versions map[string]string, exclude map[string]string, hashes map[string][]string) string {
	names := make([]string, 0, len(versions))
	for name := range versions {
		if _, skip := exclude[name]; !skip {
			names = append(names, name)
		}
	}
	sort.Strings(names)

	var b strings.Builder
	for _, name := range names {
		b.WriteString(name + "==" + versions[name])
		for _, hash := range hashes[name] {
			b.WriteString(" \\\n    --hash=" + hash)
		}
		b.WriteString("\n")
	}
	return b.String()
}

const HashLookups = 8

func (p Project) RequirementsHeader() string {
	return Render("requirements_header.txt.tmpl", struct{ Python, OS, Arch string }{p.Resolved.PythonVersion, runtime.GOOS, runtime.GOARCH})
}
