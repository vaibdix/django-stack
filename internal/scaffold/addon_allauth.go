package scaffold

import (
	"context"
	"path/filepath"

	"github.com/vaibdix/django-stack/internal/assets"
	"github.com/vaibdix/django-stack/internal/config"
)

func (e *Env) applyAllAuth(ctx context.Context) error {
	if e.cfg.CreateTemplates() {
		if err := e.emit(e.allAuthFiles()); err != nil {
			return err
		}

		e.Step("Created styled AllAuth templates (login, signup, logout).")
	}

	return nil
}

func (e *Env) allAuthFiles() fileSet {
	files := fileSet{}
	if e.cfg.Has(config.AllAuth) && e.cfg.CreateTemplates() {
		files.addAll(filepath.Join(templatesDir, "account"), assets.AllAuthTemplates(e.cfg.UsesTailwind()))
	}
	return files
}

func allAuthAddOn() addOn {
	return addOn{
		enabled: has(config.AllAuth),
		steps:   []phasedStep{{phaseFeatures, step{name: "setup allauth", label: "Setting up Django AllAuth...", run: (*Env).applyAllAuth}}},
		contribute: func(*Env) contribution {
			return contribution{
				runtime: []string{pkgAllAuth},
				readme:  func(d *readmeData) { d.AllAuth, d.DjangoAuth = true, false },
				settings: settingsContribution{
					apps:       []string{appAllAuth, appAllAuthAccount},
					middleware: []string{middlewareAllAuth},
					blocks:     []string{allauthSettingsBlock()},
				},
				urls: urlContribution{
					patterns: []string{"    path('accounts/', include('allauth.urls')),"},
					auth:     true,
				},
			}
		},
	}
}

func allauthSettingsBlock() string { return renderStatic("allauth.py.tmpl") }
