package scaffold

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/vaibdix/django-stack/internal/assets"
)

func (e *Env) applyAllAuth(ctx context.Context) error {
	settingsPath := filepath.Join(e.dir, e.cfg.ModuleName(), settingsFile)
	se, err := newSettingsEditor(settingsPath)
	if err != nil {
		return err
	}

	appsToAdd := []string{appAllAuth, appAllAuthAccount}
	for _, app := range appsToAdd {
		if err := se.AddToList(installedApps, app); err != nil {
			return fmt.Errorf("failed to add %s to %s: %w", app, installedApps, err)
		}
	}

	if err := se.AddToList(middlewareList, middlewareAllAuth); err != nil {
		return fmt.Errorf("failed to add %s to %s: %w", middlewareAllAuth, middlewareList, err)
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
		accountTemplatesPath := filepath.Join(e.dir, templatesDir, "account")
		if err := os.MkdirAll(accountTemplatesPath, 0755); err != nil {
			return fmt.Errorf("failed to create templates/account directory: %w", err)
		}

		authTemplates := assets.AllAuthTemplates(e.cfg.UsesTailwind())
		for name, content := range authTemplates {
			if err := os.WriteFile(filepath.Join(accountTemplatesPath, name), []byte(content), 0644); err != nil {
				return fmt.Errorf("failed to create %s: %w", name, err)
			}
		}

		e.Step("Created styled AllAuth templates (login, signup, logout).")
	}

	return nil
}
