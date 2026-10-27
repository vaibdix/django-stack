package render

import (
	"path/filepath"
	"strings"

	"github.com/vaibdix/django-stack/internal/config"
)

func (p Project) mypySection() string {
	metadata := p.PythonToolingMetadata()
	plugins := []string{`"mypy_django_plugin.main"`}
	if p.Cfg.Has(config.RestFramework) {
		plugins = append(plugins, `"mypy_drf_plugin.main"`)
	}
	if p.usesPydanticModels() {
		plugins = append(plugins, `"pydantic.mypy"`)
	}

	settingsModule := p.typecheckSettingsName()

	return Render("pyproject_mypy.toml.tmpl", struct{ PythonVersion, Plugins, SettingsModule, Pydantic, Untyped string }{metadata.MypyPythonVersion, strings.Join(plugins, ", "), settingsModule, p.pydanticMypySection(), p.untypedImportsOverride()})
}

func (p Project) typecheckSettingsName() string {
	module := p.Cfg.ModuleName()
	if p.Cfg.Has(config.MultiEnv) {
		return module + ".settings.test"
	}
	return module + "." + strings.TrimSuffix(typecheckSettingsFile, ".py")
}

func (p Project) typecheckFiles() FileSet {
	files := FileSet{}
	if p.Cfg.Has(config.Mypy) && !p.Cfg.Has(config.MultiEnv) {
		files.add(filepath.Join(p.Cfg.ModuleName(), typecheckSettingsFile), typecheckSettingsSource(), FileMode)
	}
	return files
}

const typecheckSettingsFile = "typecheck_settings.py"

func typecheckSettingsSource() string { return RenderStatic("typecheck_settings.py.tmpl") }

func (p Project) usesPydanticModels() bool {
	return p.Cfg.Has(config.Pydantic) || p.Cfg.Has(config.Ninja)
}

func (p Project) pydanticMypySection() string {
	if !p.usesPydanticModels() {
		return ""
	}
	return RenderStatic("pyproject_mypy_pydantic.toml.tmpl")
}

func (p Project) untypedImportsOverride() string {
	var modules []string
	if !p.Cfg.Has(config.Pydantic) {
		modules = append(modules, `"environ"`)
	}
	if p.Cfg.Has(config.AllAuth) {
		modules = append(modules, `"allauth.*"`)
	}
	if len(modules) == 0 {
		return ""
	}
	return Render("pyproject_mypy_untyped.toml.tmpl", struct{ Modules string }{strings.Join(modules, ", ")})
}

func mypyFeature() feature {
	return feature{
		enabled: Has(config.Mypy),
		contribute: func(p Project) contribution {
			stubs, stubsExt := djangoStubsSpecs(p.Cfg.Django().String())
			dev := []string{PkgMypy, stubs}
			if p.Cfg.Has(config.RestFramework) {
				dev = append(dev, PkgDRFStubs)
			}
			return contribution{
				runtime: []string{stubsExt},
				dev:     dev,
				readme:  func(d *readmeData) { d.Mypy = true },
				checks:  []VerificationCheck{{Name: "mypy check", Args: []string{"-m", "mypy", "."}}},
				settings: settingsContribution{
					imports: []string{"import django_stubs_ext"},
					blocks:  []block{{rank: rankMypy, text: "\ndjango_stubs_ext.monkeypatch()\n"}},
				},
			}
		},
	}
}

func (p Project) typedAgainstStubs52() bool {
	return p.Cfg.Has(config.Mypy) && p.Cfg.Django().Series() == "5.2"
}
