package scaffold

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/vaibdix/django-cli/internal/assets"
)

func (e *Env) applyAllAuth(projectPath string) error {
	if !e.Cfg.SetupAllAuth {
		return nil
	}

	settingsPath := filepath.Join(projectPath, e.Cfg.ModuleName(), SettingsFile)
	se, err := NewSettingsEditor(settingsPath)
	if err != nil {
		return err
	}

	appsToAdd := []string{AppContribSites, AppAllAuth, AppAllAuthAccount}
	for _, app := range appsToAdd {
		if !se.Contains("'" + app + "',") {
			if err := se.AddToList(InstalledApps, app); err != nil {
				return fmt.Errorf("failed to add %s to %s: %w", app, InstalledApps, err)
			}
		}
	}

	if !se.Contains(MiddlewareAllAuth) {
		if err := se.AddToList(MiddlewareList, MiddlewareAllAuth); err != nil {
			return fmt.Errorf("failed to add %s to %s: %w", MiddlewareAllAuth, MiddlewareList, err)
		}
	}

	allauthSettings := `

AUTHENTICATION_BACKENDS = [
    'django.contrib.auth.backends.ModelBackend',
    'allauth.account.auth_backends.AuthenticationBackend',
]

SITE_ID = 1

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

	if e.Cfg.CreateTemplates {
		accountTemplatesPath := filepath.Join(projectPath, TemplatesDir, "account")
		if err := os.MkdirAll(accountTemplatesPath, 0755); err != nil {
			return fmt.Errorf("failed to create templates/account directory: %w", err)
		}

		authTemplates := map[string]string{
			"logout.html": assets.AllAuthLogout(e.Cfg.SetupTailwind),
		}
		for _, page := range assets.AuthPages {
			authTemplates[page.Name+".html"] = assets.AuthFormTemplate(page, e.Cfg.SetupTailwind)
		}
		for _, page := range assets.AuthNotices {
			authTemplates[page.Name+".html"] = assets.AuthNoticeTemplate(page, e.Cfg.SetupTailwind)
		}
		for name, content := range authTemplates {
			if err := os.WriteFile(filepath.Join(accountTemplatesPath, name), []byte(content), 0644); err != nil {
				return fmt.Errorf("failed to create %s: %w", name, err)
			}
		}

		e.Step("Created styled AllAuth templates (login, signup, logout).")
	}

	return nil
}
