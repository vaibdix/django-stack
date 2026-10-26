package scaffold

import (
	"path/filepath"

	"github.com/vaibdix/django-stack/internal/config"
)

func (e *Env) apiFiles() fileSet {
	files := fileSet{}
	switch {
	case e.cfg.App == "":
	case e.cfg.Has(config.RestFramework):
		files.addAll(filepath.Join(e.cfg.App, "api"), generateAPIPackage(e.cfg.App))
	case e.cfg.Has(config.Ninja):
		files.addAll(filepath.Join(e.cfg.App, "api"), generateNinjaPackage(e.cfg.App, e.cfg.Name))
	}
	return files
}

func (e *Env) writeAPIPackage() error {
	if err := e.emit(e.apiFiles()); err != nil {
		return err
	}
	e.Step("Created api/ package.")
	return nil
}

type apiPackageData struct {
	AppName string
}

func generateAPIPackage(appName string) map[string]string {
	data := apiPackageData{AppName: appName}
	return map[string]string{
		"__init__.py":    render("api_init.py.tmpl", data),
		"serializers.py": render("api_serializers.py.tmpl", data),
		"views.py":       render("api_views.py.tmpl", data),
		"urls.py":        render("api_urls.py.tmpl", data),
	}
}
