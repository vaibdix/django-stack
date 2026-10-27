package render

import (
	"path/filepath"

	"github.com/vaibdix/django-stack/internal/config"
)

func (p Project) apiFiles() FileSet {
	files := FileSet{}
	switch {
	case !p.Cfg.App().Ok():
	case p.Cfg.Has(config.RestFramework):
		files.addAll(filepath.Join(p.Cfg.App().String(), "api"), apiPackage(apiPackageData{AppName: p.Cfg.App().String(), Lazy: p.lazyAnnotations()}))
	case p.Cfg.Has(config.Ninja):
		files.addAll(filepath.Join(p.Cfg.App().String(), "api"), generateNinjaPackage(p.Cfg.App().String(), p.Cfg.Name().String()))
	}
	return files
}

type apiPackageData struct {
	AppName string
	Lazy    bool
}

func apiPackage(data apiPackageData) map[string]string {
	return map[string]string{
		"__init__.py":    Render("api_init.py.tmpl", data),
		"serializers.py": Render("api_serializers.py.tmpl", data),
		"views.py":       Render("api_views.py.tmpl", data),
		"urls.py":        Render("api_urls.py.tmpl", data),
	}
}
