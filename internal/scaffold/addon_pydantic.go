package scaffold

import (
	"path/filepath"
	"strconv"

	"github.com/vaibdix/django-stack/internal/config"
)

func pydanticFeature() feature {
	return feature{
		enabled: has(config.Pydantic),
		contribute: func(project) contribution {
			return contribution{readme: func(d *readmeData) { d.Pydantic = true }}
		},
	}
}

func (p project) configPackages() []string {
	if p.cfg.Has(config.Pydantic) {
		return []string{pkgPydantic, pkgPydanticSettings}
	}
	return []string{pkgEnviron}
}

func (p project) envConfig() envConfig {
	return envConfig{
		Database:   p.cfg.Database(),
		Pydantic:   p.cfg.Has(config.Pydantic),
		ModuleName: p.cfg.ModuleName(),

		SettingsPackage: p.split(),
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

func (p project) confTemplateData() confTemplateData {
	data := confTemplateData{Sentry: p.cfg.Has(config.Sentry), Ruff: p.cfg.Has(config.Ruff)}

	if p.cfg.Database().UsesDocker() {
		database := p.cfg.Database().Spec()
		data.Database, data.DBUser, data.DBHost, data.DBPort = true, "django_user", database.Host, strconv.Itoa(database.Port)
	}

	return data
}

func (p project) confFiles() fileSet {
	files := fileSet{}
	if p.cfg.Has(config.Pydantic) {
		files.add(filepath.Join(p.cfg.ModuleName(), confFile), render("conf.py.tmpl", p.confTemplateData()), fileMode)
	}
	return files
}
