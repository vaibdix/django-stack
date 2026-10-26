package scaffold

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/vaibdix/django-stack/internal/assets"
	"github.com/vaibdix/django-stack/internal/config"
)

func (e *Env) applyAllAuth(ctx context.Context) error {
	se, err := e.settings()
	if err != nil {
		return err
	}

	appsToAdd := []string{appAllAuth, appAllAuthAccount}
	for _, app := range appsToAdd {
		if err := se.AddToList(installedApps, app); err != nil {
			return fmt.Errorf("add %s to %s: %w", app, installedApps, err)
		}
	}

	if err := se.AddToList(middlewareList, middlewareAllAuth); err != nil {
		return fmt.Errorf("add %s to %s: %w", middlewareAllAuth, middlewareList, err)
	}

	allauthSettings := `

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
	se.AppendBlock(allauthSettings)
	se.SetTopLevel("LOGIN_URL", "'account_login'")

	if err := se.Save(); err != nil {
		return err
	}
	e.Step("Configured Django AllAuth: INSTALLED_APPS, MIDDLEWARE, AUTHENTICATION_BACKENDS, and account settings.")

	if e.cfg.CreateTemplates() {
		if err := e.writeFiles(filepath.Join(templatesDir, "account"), assets.AllAuthTemplates(e.cfg.UsesTailwind())); err != nil {
			return err
		}

		e.Step("Created styled AllAuth templates (login, signup, logout).")
	}

	return nil
}

func allAuthAddOn() addOn {
	return addOn{
		enabled: has(config.AllAuth),
		steps:   []phasedStep{{phaseFeatures, step{name: "setup allauth", label: "Setting up Django AllAuth...", run: (*Env).applyAllAuth}}},
		contribute: func(*Env) contribution {
			return contribution{
				runtime: []string{pkgAllAuth},
				readme:  func(d *readmeData) { d.AllAuth, d.DjangoAuth = true, false },
				urls: urlContribution{
					patterns: []string{"    path('accounts/', include('allauth.urls')),"},
					auth:     true,
				},
			}
		},
	}
}
