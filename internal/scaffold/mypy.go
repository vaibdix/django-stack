package scaffold

import "fmt"

func (e *Env) applyMypy(projectPath string) error {
	packages := []string{PkgMypy, PkgDjangoStubs}
	label := "mypy + django-stubs"

	if e.Cfg.SetupRestFramework {
		packages = append(packages, PkgDRFStubs)
		label = "mypy + django-stubs + drf-stubs"
	}

	if err := e.installPackages(projectPath, label, packages...); err != nil {
		return err
	}

	metadata := e.pythonToolingMetadata()
	pe := NewPyprojectEditor(projectPath, e.Cfg.ProjectName, metadata)

	if !pe.Contains("[tool.mypy]") {
		plugins := `["mypy_django_plugin.main"]`
		if e.Cfg.SetupRestFramework {
			plugins = `["mypy_django_plugin.main", "mypy_drf_plugin.main"]`
		}

		settingsModule := fmt.Sprintf("%s.settings", e.Cfg.ModuleName())
		if e.Cfg.SetupMultiEnv {
			settingsModule = fmt.Sprintf("%s.settings.local", e.Cfg.ModuleName())
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

	e.Step("Mypy type checking configured in pyproject.toml")
	return nil
}
