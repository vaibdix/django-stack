package scaffold

import (
	"path/filepath"

	"github.com/vaibdix/django-stack/internal/config"
)

func (p project) apiFiles() fileSet {
	files := fileSet{}
	switch {
	case !p.cfg.App().Ok():
	case p.cfg.Has(config.RestFramework):
		files.addAll(filepath.Join(p.cfg.App().String(), "api"), apiPackage(apiPackageData{AppName: p.cfg.App().String(), Lazy: p.lazyAnnotations()}))
	case p.cfg.Has(config.Ninja):
		files.addAll(filepath.Join(p.cfg.App().String(), "api"), generateNinjaPackage(p.cfg.App().String(), p.cfg.Name().String()))
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
	Lazy    bool
}

func apiPackage(data apiPackageData) map[string]string {
	return map[string]string{
		"__init__.py":    render("api_init.py.tmpl", data),
		"serializers.py": render("api_serializers.py.tmpl", data),
		"views.py":       render("api_views.py.tmpl", data),
		"urls.py":        render("api_urls.py.tmpl", data),
	}
}
