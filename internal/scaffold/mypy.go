package scaffold

import (
	"fmt"
	"strings"
)

func (e *Env) applyMypy(projectPath string) error {
	packages := []string{PkgMypy, PkgDjangoStubs}
	label := "mypy + django-stubs"

	if e.Cfg.SetupRestFramework {
		packages = append(packages, PkgDRFStubs)
		label = "mypy + django-stubs + drf-stubs"
	}

	if err := e.installDevPackages(projectPath, label, packages...); err != nil {
		return err
	}

	// django-stubs-ext is a runtime dependency, unlike django-stubs itself:
	// settings calls its monkeypatch so that ModelForm[Book] is subscriptable
	// at import time. Without it a parameterised form raises TypeError and the
	// project does not start.
	if err := e.installPackages(projectPath, PkgDjangoStubsExt, PkgDjangoStubsExt); err != nil {
		return err
	}
	if err := e.enableRuntimeGenerics(projectPath); err != nil {
		return err
	}

	metadata := e.pythonToolingMetadata()
	pe := NewPyprojectEditor(projectPath, e.Cfg.ProjectName, metadata)

	if !pe.Contains("[tool.mypy]") {
		plugins := []string{`"mypy_django_plugin.main"`}
		if e.Cfg.SetupRestFramework {
			plugins = append(plugins, `"mypy_drf_plugin.main"`)
		}
		if e.Cfg.SetupPydanticConf {
			plugins = append(plugins, `"pydantic.mypy"`)
		}

		settingsModule := fmt.Sprintf("%s.settings", e.Cfg.ModuleName())
		if e.Cfg.SetupMultiEnv {
			settingsModule = fmt.Sprintf("%s.settings.local", e.Cfg.ModuleName())
		}

		section := fmt.Sprintf(`[tool.mypy]
python_version = "%s"
strict = true
# strict does not ban explicit Any; this does. Nothing this project generates
# needs one — object, TypedDict or a pydantic model says more in every case.
disallow_any_explicit = true
disallow_any_unimported = true
warn_unreachable = true
ignore_missing_imports = true
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
`, metadata.MypyPythonVersion, strings.Join(plugins, ", "), settingsModule, e.pydanticMypySection())

		pe.AppendSection(section)
	}

	if err := pe.Save(); err != nil {
		return err
	}

	e.Step("Mypy type checking configured in pyproject.toml (strict, no Any)")
	return nil
}

// enableRuntimeGenerics makes django.forms.ModelForm and admin.ModelAdmin
// subscriptable. Django does not define __class_getitem__ on them, so
// ModelForm[Book] is a TypeError until django_stubs_ext patches it in. This has
// to run in settings, which is imported before any form or admin module.
func (e *Env) enableRuntimeGenerics(projectPath string) error {
	se, err := e.settingsEditor(projectPath)
	if err != nil {
		return err
	}
	if se.Contains("django_stubs_ext.monkeypatch()") {
		return nil
	}

	se.AddImport("import django_stubs_ext")
	se.AppendBlock(`
# Django's ModelForm and ModelAdmin are generic to type checkers but not at
# runtime. This makes ModelForm[Book] a real expression rather than a TypeError,
# and it must run before any form or admin module is imported.
django_stubs_ext.monkeypatch()
`)

	if err := se.Save(); err != nil {
		return err
	}
	e.Step("Enabled runtime generics so ModelForm[Book] and ModelAdmin[Book] resolve.")
	return nil
}

// pydanticMypySection switches off the one Any this project cannot annotate away.
// The pydantic plugin appends "**kwargs: Any" to the __init__ it synthesises for
// every model, but only while init_forbid_extra is off — so this is what lets
// conf.py keep extra='ignore' at runtime and still satisfy disallow_any_explicit.
func (e *Env) pydanticMypySection() string {
	if !e.Cfg.SetupPydanticConf {
		return ""
	}
	return `
[tool.pydantic-mypy]
init_typed = true
init_forbid_extra = true
warn_required_dynamic_aliases = true
`
}
