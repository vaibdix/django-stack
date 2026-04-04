package ui

import "fmt"

// SetupMypyTypeChecking installs mypy with django-stubs (and optionally
// djangorestframework-stubs), then writes mypy configuration to pyproject.toml.
func (m *Model) SetupMypyTypeChecking(projectPath string) error {
	m.UpdateProgress("Setting up Mypy type checking...")

	packages := []string{PkgMypy, PkgDjangoStubs}
	label := "mypy + django-stubs"

	if m.SetupRestFramework {
		packages = append(packages, PkgDRFStubs)
		label = "mypy + django-stubs + drf-stubs"
	}

	if err := m.installAndReport(projectPath, label, packages...); err != nil {
		return err
	}

	metadata := m.pythonToolingMetadata()
	pe := NewPyprojectEditor(projectPath, m.ProjectName, metadata)

	if !pe.Contains("[tool.mypy]") {
		plugins := `["mypy_django_plugin.main"]`
		if m.SetupRestFramework {
			plugins = `["mypy_django_plugin.main", "mypy_drf_plugin.main"]`
		}

		settingsModule := fmt.Sprintf("%s.settings", m.ProjectModuleName())
		if m.SetupMultiEnv {
			settingsModule = fmt.Sprintf("%s.settings.local", m.ProjectModuleName())
		}

		section := fmt.Sprintf(`[tool.mypy]
python_version = "%s"
check_untyped_defs = true
ignore_missing_imports = true
warn_unused_ignores = true
warn_redundant_casts = true
plugins = %s

[tool.django-stubs]
django_settings_module = "%s"
`, metadata.MypyPythonVersion, plugins, settingsModule)

		pe.AppendSection(section)
	}

	if err := pe.Save(); err != nil {
		return err
	}

	m.AppendStepMessage("Mypy type checking configured in pyproject.toml")
	return nil
}
