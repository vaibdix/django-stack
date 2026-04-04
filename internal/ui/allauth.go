package ui

import (
	"fmt"
	"os"
	"path/filepath"

	"django-stack/internal/templates"
)

func (m *Model) SetupAllAuthIntegration(projectPath string) error {
	if !m.SetupAllAuth {
		return nil
	}
	m.UpdateProgress("Setting up Django AllAuth...")

	if err := m.installAndReport(projectPath, "django-allauth", PkgAllAuth); err != nil {
		return err
	}

	settingsPath := filepath.Join(projectPath, m.ProjectModuleName(), SettingsFile)
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

ACCOUNT_EMAIL_VERIFICATION = 'none'

ACCOUNT_AUTHENTICATION_METHOD = 'username_email'

ACCOUNT_SIGNUP_FIELDS = ['email*', 'username*', 'password1*', 'password2*']
`
	se.AppendBlock(allauthSettings)

	if err := se.Save(); err != nil {
		return err
	}
	m.AppendStepMessage("Configured Django AllAuth: INSTALLED_APPS, MIDDLEWARE, AUTHENTICATION_BACKENDS, and account settings.")

	if m.CreateTemplates {
		accountTemplatesPath := filepath.Join(projectPath, TemplatesDir, "account")
		if err := os.MkdirAll(accountTemplatesPath, 0755); err != nil {
			return fmt.Errorf("failed to create templates/account directory: %w", err)
		}

		authTemplates := map[string]string{
			"login.html":  templates.GetAllAuthLoginTemplate(m.SetupTailwind),
			"signup.html": templates.GetAllAuthSignupTemplate(m.SetupTailwind),
			"logout.html": templates.GetAllAuthLogoutTemplate(m.SetupTailwind),
		}
		for name, content := range authTemplates {
			if err := os.WriteFile(filepath.Join(accountTemplatesPath, name), []byte(content), 0644); err != nil {
				return fmt.Errorf("failed to create %s: %w", name, err)
			}
		}

		m.AppendStepMessage("Created styled AllAuth templates (login, signup, logout).")
	}

	return nil
}
