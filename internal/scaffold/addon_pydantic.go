package scaffold

import (
	"path/filepath"
	"strconv"

	"github.com/vaibdix/django-stack/internal/config"
)

func pydanticAddOn() addOn {
	return addOn{
		enabled: has(config.Pydantic),
		contribute: func(*Env) contribution {
			return contribution{readme: func(d *readmeData) { d.Pydantic = true }}
		},
	}
}

func (e *Env) configPackages() []string {
	if e.cfg.Has(config.Pydantic) {
		return []string{pkgPydantic, pkgPydanticSettings}
	}
	return []string{pkgEnviron}
}

func (e *Env) envConfig() envConfig {
	return envConfig{
		Database:   e.cfg.Database,
		Pydantic:   e.cfg.Has(config.Pydantic),
		ModuleName: e.cfg.ModuleName(),
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

func (e *Env) confTemplateData() confTemplateData {
	data := confTemplateData{Sentry: e.cfg.Has(config.Sentry), Ruff: e.cfg.Has(config.Ruff)}

	if e.cfg.Database.UsesDocker() {
		database := e.cfg.Database.Spec()
		data.Database, data.DBUser, data.DBHost, data.DBPort = true, "django_user", database.Host, strconv.Itoa(database.Port)
	}

	return data
}

func (e *Env) confFiles() fileSet {
	files := fileSet{}
	if e.cfg.Has(config.Pydantic) {
		files.add(filepath.Join(e.cfg.ModuleName(), confFile), render("conf.py.tmpl", e.confTemplateData()), fileMode)
	}
	return files
}

func (e *Env) writeConfModule() error {
	if !e.cfg.Has(config.Pydantic) {
		return nil
	}

	if err := e.emit(e.confFiles()); err != nil {
		return err
	}

	e.Step("Created: " + e.cfg.ModuleName() + "/" + confFile + " (typed settings validated at startup)")
	return nil
}
