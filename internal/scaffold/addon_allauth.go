package scaffold

import (
	"context"
	"path/filepath"

	"github.com/vaibdix/django-stack/internal/assets"
	"github.com/vaibdix/django-stack/internal/config"
)

func (e *Env) applyAllAuth(ctx context.Context) error {
	if e.cfg.CreateTemplates() {
		if err := e.emitGroup(groupAllAuth); err != nil {
			return err
		}

		e.Step("Created styled AllAuth templates (login, signup, logout).")
	}

	return nil
}

func (p project) allAuthFiles() fileSet {
	files := fileSet{}
	if p.cfg.Has(config.AllAuth) && p.cfg.CreateTemplates() {
		files.addAll(filepath.Join(templatesDir, "account"), assets.AllAuthTemplates(p.cfg.UsesTailwind()))
	}
	return files
}

func allAuthFeature() feature {
	return feature{
		enabled: has(config.AllAuth),
		contribute: func(project) contribution {
			return contribution{
				runtime: []string{pkgAllAuth},
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

func allauthSettingsBlock() string { return renderStatic("allauth.py.tmpl") }
