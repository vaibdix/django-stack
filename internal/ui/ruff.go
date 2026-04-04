package ui

import (
	"fmt"
	"strings"
)

// SetupRuffLinting installs ruff and djlint, then writes ruff configuration
// to pyproject.toml.
func (m *Model) SetupRuffLinting(projectPath string) error {
	m.UpdateProgress("Setting up Ruff linter...")

	metadata := m.pythonToolingMetadata()
	pe := NewPyprojectEditor(projectPath, m.ProjectName, metadata)

	if !pe.Contains("[tool.ruff]") {
		firstParty := []string{fmt.Sprintf(`"%s"`, m.ProjectModuleName())}
		if m.AppName != "" {
			firstParty = append(firstParty, fmt.Sprintf(`"%s"`, m.AppName))
		}

		pe.AppendSection(fmt.Sprintf(`[tool.ruff]
target-version = "%s"
line-length = 120

[tool.ruff.lint]
select = [
    "A",
    "B",
    "C4",
    "DJ",
    "E",
    "F",
    "I",
    "S",
    "SIM",
    "UP",
    "W",
]
ignore = ["RUF012", "S101", "SIM102"]

[tool.ruff.lint.isort]
force-single-line = true
known-first-party = [%s]
known-third-party = ["django", "rest_framework", "drf_spectacular", "allauth", "django_htmx", "django_browser_reload", "environ", "dotenv"]
`, metadata.RuffTargetVersion, strings.Join(firstParty, ", ")))
	}

	if err := pe.Save(); err != nil {
		return err
	}

	m.AppendStepMessage("Ruff linter configured in pyproject.toml")
	return nil
}
