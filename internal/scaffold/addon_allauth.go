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
					blocks:     []string{allauthSettingsBlock},
				},
				urls: urlContribution{
					patterns: []string{"    path('accounts/', include('allauth.urls')),"},
					auth:     true,
				},
			}
		},
	}
}

const allauthSettingsBlock = `

AUTHENTICATION_BACKENDS = [
    'django.contrib.auth.backends.ModelBackend',
    'allauth.account.auth_backends.AuthenticationBackend',
]

LOGIN_REDIRECT_URL = '/'

ACCOUNT_LOGOUT_REDIRECT_URL = '/'

ACCOUNT_EMAIL_VERIFICATION = 'none' if DEBUG else 'mandatory'

ACCOUNT_LOGIN_METHODS = {'email', 'username'}

ACCOUNT_SIGNUP_FIELDS = ['email*', 'username*', 'password1*', 'password2*']
`
