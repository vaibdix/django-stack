package render

import (
	"path/filepath"
	"strconv"

	"github.com/vaibdix/django-stack/internal/config"
)

func pydanticFeature() feature {
	return feature{
		enabled: Has(config.Pydantic),
		contribute: func(Project) contribution {
			return contribution{readme: func(d *readmeData) { d.Pydantic = true }}
		},
	}
}

func (p Project) configPackages() []string {
	if p.Cfg.Has(config.Pydantic) {
		return []string{pkgPydantic, pkgPydanticSettings}
	}
	return []string{PkgEnviron}
}

func (p Project) envConfig() envConfig {
	return envConfig{
		Database:   p.Cfg.Database(),
		Pydantic:   p.Cfg.Has(config.Pydantic),
		ModuleName: p.Cfg.ModuleName(),

		SettingsPackage: p.Split(),
	}
}

type confTemplateData struct {
	Database bool
	DBUser   string
	DBHost   string
	DBPort   string
	Sentry   bool
	Ruff     bool
}

func (confTemplateData) TestSecretKey() string { return testSecretKey }

func (p Project) confTemplateData() confTemplateData {
	data := confTemplateData{Sentry: p.Cfg.Has(config.Sentry), Ruff: p.Cfg.Has(config.Ruff)}

	if p.Cfg.Database().UsesDocker() {
		database := p.Cfg.Database().Spec()
		data.Database, data.DBUser, data.DBHost, data.DBPort = true, "django_user", database.Host, strconv.Itoa(database.Port)
	}

	return data
}

func (p Project) ConfFiles() FileSet {
	files := FileSet{}
	if p.Cfg.Has(config.Pydantic) {
		files.add(filepath.Join(p.Cfg.ModuleName(), ConfFile), Render("conf.py.tmpl", p.confTemplateData()), FileMode)
	}
	return files
}
