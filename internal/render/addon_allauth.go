package render

import (
	"path/filepath"

	"github.com/vaibdix/django-stack/internal/assets"
	"github.com/vaibdix/django-stack/internal/config"
)

func (p Project) allAuthFiles() FileSet {
	files := FileSet{}
	if p.Cfg.Has(config.AllAuth) && p.Cfg.CreateTemplates() {
		files.addAll(filepath.Join(templatesDir, "account"), assets.AllAuthTemplates(p.Cfg.UsesTailwind()))
	}
	return files
}

func allAuthFeature() feature {
	return feature{
		enabled: Has(config.AllAuth),
		contribute: func(Project) contribution {
			return contribution{
				runtime: []string{PkgAllAuth},
				readme:  func(d *readmeData) { d.AllAuth, d.DjangoAuth = true, false },
				settings: settingsContribution{
					apps:       []entry{{name: appAllAuth, after: lastDjangoApp, rank: rankAllAuth}, {name: appAllAuthAccount, after: appAllAuth}},
					middleware: []entry{{name: middlewareAllAuth, after: lastDjangoMiddleware, rank: rankAllAuth}},
					blocks:     []block{{rank: rankAllAuth, text: allauthSettingsBlock()}},
				},
				urls: urlContribution{
					routes: []route{{rank: rankAllAuthRoute, line: `    path("accounts/", include("allauth.urls")),`}},
					auth:   true,
				},
			}
		},
	}
}

func allauthSettingsBlock() string { return RenderStatic("allauth.py.tmpl") }
