package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	AccountsApp = "accounts"
	CoreApp     = "core"
)

func (e *Env) createAccountsApp(projectPath string) error {
	appDir := filepath.Join(projectPath, AccountsApp)
	migrationsDir := filepath.Join(appDir, "migrations")
	if err := os.MkdirAll(migrationsDir, 0755); err != nil {
		return fmt.Errorf("failed to create the accounts app: %w", err)
	}

	files := map[string]string{
		filepath.Join(appDir, "__init__.py"):        "",
		filepath.Join(migrationsDir, "__init__.py"): "",
		filepath.Join(appDir, "models.py"):          render("accounts_models.py.tmpl", struct{ Stubs52 bool }{e.typedAgainstStubs52()}),
		filepath.Join(appDir, "apps.py"):            renderStatic("accounts_apps.py.tmpl"),
		filepath.Join(appDir, "admin.py"):           generateAccountsAdmin(typingData{RuntimeGenerics: e.Cfg.SetupMypy}),
		filepath.Join(appDir, "tests.py"): render("accounts_tests.py.tmpl", struct{ AllAuthPages bool }{
			e.Cfg.SetupAllAuth && e.Cfg.CreateTemplates,
		}),
	}
	for path, content := range files {
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", filepath.Base(path), err)
		}
	}

	coreDir := filepath.Join(projectPath, CoreApp)
	if err := os.MkdirAll(coreDir, 0755); err != nil {
		return fmt.Errorf("failed to create the core app: %w", err)
	}
	coreFiles := map[string]string{
		filepath.Join(coreDir, "__init__.py"): "",
		filepath.Join(coreDir, "models.py"):   renderStatic("core_models.py.tmpl"),
		filepath.Join(coreDir, "apps.py"):     renderStatic("core_apps.py.tmpl"),
	}
	for path, content := range coreFiles {
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", filepath.Base(path), err)
		}
	}

	se, err := e.settingsEditor(projectPath)
	if err != nil {
		return err
	}
	if err := se.AddToList(InstalledApps, CoreApp); err != nil {
		return fmt.Errorf("failed to register the core app: %w", err)
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

	e.Step("Created the core and accounts apps; the custom User model lands before the first migration.")
	return nil
}

func generateAccountsAdmin(data typingData) string {
	return render("accounts_admin.py.tmpl", data)
}
