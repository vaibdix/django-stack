package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
)

const AccountsApp = "accounts"

func (e *Env) createAccountsApp(projectPath string) error {
	appDir := filepath.Join(projectPath, AccountsApp)
	migrationsDir := filepath.Join(appDir, "migrations")
	if err := os.MkdirAll(migrationsDir, 0755); err != nil {
		return fmt.Errorf("failed to create the accounts app: %w", err)
	}

	files := map[string]string{
		filepath.Join(appDir, "__init__.py"):        "",
		filepath.Join(migrationsDir, "__init__.py"): "",
		filepath.Join(appDir, "models.py"):          renderStatic("accounts_models.py.tmpl"),
		filepath.Join(appDir, "apps.py"):            renderStatic("accounts_apps.py.tmpl"),
		filepath.Join(appDir, "admin.py"):           renderStatic("accounts_admin.py.tmpl"),
		filepath.Join(appDir, "tests.py"):           renderStatic("accounts_tests.py.tmpl"),
	}
	for path, content := range files {
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", filepath.Base(path), err)
		}
	}

	se, err := e.settingsEditor(projectPath)
	if err != nil {
		return err
	}
	if err := se.AddToList(InstalledApps, AccountsApp); err != nil {
		return fmt.Errorf("failed to register the accounts app: %w", err)
	}
	if !se.Contains("AUTH_USER_MODEL") {
		se.AppendBlock("\nAUTH_USER_MODEL = 'accounts.User'\n")
	}
	if err := se.Save(); err != nil {
		return err
	}

	e.Step("Created the accounts app with a custom User model, before the first migration.")
	return nil
}
