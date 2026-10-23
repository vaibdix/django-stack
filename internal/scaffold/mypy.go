package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vaibdix/django-stack/internal/config"
)

func (e *Env) applyMypy(projectPath string) error {
	if err := e.enableRuntimeGenerics(projectPath); err != nil {
		return err
	}

	metadata := e.pythonToolingMetadata()
	pe, err := newPyprojectEditor(projectPath, e.cfg.Name, metadata)
	if err != nil {
		return err
	}

	if !pe.Contains("[tool.mypy]") {
		plugins := []string{`"mypy_django_plugin.main"`}
		if e.cfg.Has(config.RestFramework) {
			plugins = append(plugins, `"mypy_drf_plugin.main"`)
		}
		if e.usesPydanticModels() {
			plugins = append(plugins, `"pydantic.mypy"`)
		}

		settingsModule, err := e.typecheckSettingsModule(projectPath)
		if err != nil {
			return err
		}

		section := fmt.Sprintf(`[tool.mypy]
python_version = "%s"
strict = true
# strict does not ban explicit Any; this does. Nothing this project generates
# needs one — object, TypedDict or a pydantic model says more in every case.
disallow_any_explicit = true
disallow_any_unimported = true
warn_unreachable = true
plugins = [%s]

[tool.django-stubs]
django_settings_module = "%s"
%s
[[tool.mypy.overrides]]
# Django writes migrations; they are not ours to type.
module = ["*.migrations.*"]
ignore_errors = true

[[tool.mypy.overrides]]
# "from .base import *" is the documented Django settings layout, and
# no_implicit_reexport reports every name it carries as undefined. Both patterns
# are listed because settings is a single module until the multi-env split
# turns it into a package.
module = ["*.settings", "*.settings.*"]
implicit_reexport = true
%s`, metadata.MypyPythonVersion, strings.Join(plugins, ", "), settingsModule, e.pydanticMypySection(), e.untypedImportsOverride())

		pe.AppendSection(section)
	}

	if err := pe.Save(); err != nil {
		return err
	}

	e.Step("Mypy type checking configured in pyproject.toml (strict, no Any)")
	return nil
}

func (e *Env) typecheckSettingsModule(projectPath string) (string, error) {
	module := e.cfg.ModuleName()
	if e.cfg.Has(config.MultiEnv) {
		return module + ".settings.test", nil
	}

	path := filepath.Join(projectPath, module, typecheckSettingsFile)
	if err := os.WriteFile(path, []byte(typecheckSettingsSource), 0644); err != nil {
		return "", fmt.Errorf("failed to write %s: %w", typecheckSettingsFile, err)
	}
	return module + "." + strings.TrimSuffix(typecheckSettingsFile, ".py"), nil
}

const typecheckSettingsFile = "typecheck_settings.py"

const typecheckSettingsSource = `import os

os.environ.setdefault('ENVIRONMENT', 'test')

from .settings import *
`

func (e *Env) enableRuntimeGenerics(projectPath string) error {
	se, err := e.settingsEditor(projectPath)
	if err != nil {
		return err
	}
	if se.Contains("django_stubs_ext.monkeypatch()") {
		return nil
	}

	if err := se.AddImport("import django_stubs_ext"); err != nil {
		return err
	}
	se.AppendBlock(`
django_stubs_ext.monkeypatch()
`)

	if err := se.Save(); err != nil {
		return err
	}
	e.Step("Enabled runtime generics so ModelForm[Book] and ModelAdmin[Book] resolve.")
	return nil
}

func (e *Env) usesPydanticModels() bool {
	return e.cfg.Has(config.Pydantic) || e.cfg.Has(config.Ninja)
}

func (e *Env) pydanticMypySection() string {
	if !e.usesPydanticModels() {
		return ""
	}
	return `
[tool.pydantic-mypy]
init_typed = true
init_forbid_extra = true
warn_required_dynamic_aliases = true
`
}

func (e *Env) untypedImportsOverride() string {
	var modules []string
	if !e.cfg.Has(config.Pydantic) {
		modules = append(modules, `"environ"`)
	}
	if e.cfg.Has(config.AllAuth) {
		modules = append(modules, `"allauth.*"`)
	}
	if len(modules) == 0 {
		return ""
	}
	return `
[[tool.mypy.overrides]]
# These packages ship no type information.
module = [` + strings.Join(modules, ", ") + `]
ignore_missing_imports = true
`
}
