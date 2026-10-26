package scaffold

import (
	"path/filepath"
	"strings"

	"github.com/vaibdix/django-stack/internal/config"
)

func (p project) mypySection() string {
	metadata := p.pythonToolingMetadata()
	plugins := []string{`"mypy_django_plugin.main"`}
	if p.cfg.Has(config.RestFramework) {
		plugins = append(plugins, `"mypy_drf_plugin.main"`)
	}
	if p.usesPydanticModels() {
		plugins = append(plugins, `"pydantic.mypy"`)
	}

	settingsModule := p.typecheckSettingsName()

	return render("pyproject_mypy.toml.tmpl", struct{ PythonVersion, Plugins, SettingsModule, Pydantic, Untyped string }{metadata.MypyPythonVersion, strings.Join(plugins, ", "), settingsModule, p.pydanticMypySection(), p.untypedImportsOverride()})
}

func (p project) typecheckSettingsName() string {
	module := p.cfg.ModuleName()
	if p.cfg.Has(config.MultiEnv) {
		return module + ".settings.test"
	}
	return module + "." + strings.TrimSuffix(typecheckSettingsFile, ".py")
}

func (p project) typecheckFiles() fileSet {
	files := fileSet{}
	if p.cfg.Has(config.Mypy) && !p.cfg.Has(config.MultiEnv) {
		files.add(filepath.Join(p.cfg.ModuleName(), typecheckSettingsFile), typecheckSettingsSource(), fileMode)
	}
	return files
}

const typecheckSettingsFile = "typecheck_settings.py"

func typecheckSettingsSource() string { return renderStatic("typecheck_settings.py.tmpl") }

func (p project) usesPydanticModels() bool {
	return p.cfg.Has(config.Pydantic) || p.cfg.Has(config.Ninja)
}

func (p project) pydanticMypySection() string {
	if !p.usesPydanticModels() {
		return ""
	}
	return renderStatic("pyproject_mypy_pydantic.toml.tmpl")
}

func (p project) untypedImportsOverride() string {
	var modules []string
	if !p.cfg.Has(config.Pydantic) {
		modules = append(modules, `"environ"`)
	}
	if p.cfg.Has(config.AllAuth) {
		modules = append(modules, `"allauth.*"`)
	}
	if len(modules) == 0 {
		return ""
	}
	return render("pyproject_mypy_untyped.toml.tmpl", struct{ Modules string }{strings.Join(modules, ", ")})
}

func mypyFeature() feature {
	return feature{
		enabled: has(config.Mypy),
		contribute: func(p project) contribution {
			stubs, stubsExt := djangoStubsSpecs(p.cfg.Django().String())
			dev := []string{pkgMypy, stubs}
			if p.cfg.Has(config.RestFramework) {
				dev = append(dev, pkgDRFStubs)
			}
			return contribution{
				runtime: []string{stubsExt},
				dev:     dev,
				readme:  func(d *readmeData) { d.Mypy = true },
				checks:  []verificationCheck{{name: "mypy check", args: []string{"-m", "mypy", "."}}},
				settings: settingsContribution{
					imports: []string{"import django_stubs_ext"},
					blocks:  []block{{rank: rankMypy, text: "\ndjango_stubs_ext.monkeypatch()\n"}},
				},
			}
		},
	}
}

func (p project) typedAgainstStubs52() bool {
	return p.cfg.Has(config.Mypy) && p.cfg.Django().Series() == "5.2"
}
