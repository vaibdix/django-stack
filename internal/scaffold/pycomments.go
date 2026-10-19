package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var keptPythonComments = []string{"#!", "# noqa", "# type:", "# -*-"}

func stripPythonComments(source string) string {
	lines := strings.Split(source, "\n")
	result := make([]string, 0, len(lines))

	i := 0
	for i < len(lines) {
		trimmed := strings.TrimSpace(lines[i])

		if quote := docstringQuote(trimmed); quote != "" {
			i = endOfDocstring(lines, i, quote)
			i = skipOneBlankLine(lines, i)
			continue
		}

		if isDroppableComment(trimmed) {
			i++
			continue
		}

		result = append(result, lines[i])
		i++
	}

	return collapseBlankRuns(strings.Join(result, "\n"))
}

func docstringQuote(trimmed string) string {
	for _, quote := range []string{`"""`, `'''`} {
		if strings.HasPrefix(trimmed, quote) {
			return quote
		}
	}
	return ""
}

func endOfDocstring(lines []string, start int, quote string) int {
	body := strings.TrimPrefix(strings.TrimSpace(lines[start]), quote)
	if strings.HasSuffix(body, quote) && body != "" {
		return start + 1
	}

	for i := start + 1; i < len(lines); i++ {
		if strings.Contains(lines[i], quote) {
			return i + 1
		}
	}
	return len(lines)
}

func skipOneBlankLine(lines []string, i int) int {
	if i < len(lines) && strings.TrimSpace(lines[i]) == "" {
		return i + 1
	}
	return i
}

func isDroppableComment(trimmed string) bool {
	if !strings.HasPrefix(trimmed, "#") {
		return false
	}
	for _, kept := range keptPythonComments {
		if strings.HasPrefix(trimmed, kept) {
			return false
		}
	}
	return true
}

func collapseBlankRuns(source string) string {
	for strings.Contains(source, "\n\n\n\n") {
		source = strings.ReplaceAll(source, "\n\n\n\n", "\n\n\n")
	}
	return strings.TrimLeft(source, "\n")
}

func (e *Env) stripCommentsFromDjangoBoilerplate(projectPath string) error {
	module := filepath.Join(projectPath, e.Cfg.ModuleName())

	targets := []string{
		filepath.Join(projectPath, ManageFile),
		filepath.Join(module, SettingsFile),
		filepath.Join(module, URLsFile),
		filepath.Join(module, "wsgi.py"),
		filepath.Join(module, "asgi.py"),
		filepath.Join(module, "__init__.py"),
	}

	for _, path := range targets {
		data, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return fmt.Errorf("failed to read %s: %w", filepath.Base(path), err)
		}

		stripped := stripPythonComments(string(data))
		if stripped == string(data) {
			continue
		}

		info, err := os.Stat(path)
		if err != nil {
			return fmt.Errorf("failed to stat %s: %w", filepath.Base(path), err)
		}
		if err := os.WriteFile(path, []byte(stripped), info.Mode().Perm()); err != nil {
			return fmt.Errorf("failed to write %s: %w", filepath.Base(path), err)
		}
	}

	return nil
}
