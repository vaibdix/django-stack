package scaffold

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/vaibdix/django-stack/internal/config"
)

const (
	accountsApp = "accounts"
	coreApp     = "core"
)

func (e *Env) createAccountsApp(ctx context.Context) error {
	files := map[string]string{
		"__init__.py": "",
		filepath.Join("migrations", "__init__.py"): "",
		"models.py": render("accounts_models.py.tmpl", struct{ Stubs52 bool }{e.typedAgainstStubs52()}),
		"apps.py":   renderStatic("accounts_apps.py.tmpl"),
		"admin.py":  generateAccountsAdmin(typingData{RuntimeGenerics: e.cfg.Has(config.Mypy)}),
		"tests.py": render("accounts_tests.py.tmpl", struct{ AllAuthPages, Ruff bool }{
			e.cfg.Has(config.AllAuth) && e.cfg.CreateTemplates(), e.cfg.Has(config.Ruff),
		}),
	}
	if err := e.writeFiles(accountsApp, files); err != nil {
		return err
	}

	coreFiles := map[string]string{
		"__init__.py": "",
		"models.py":   renderStatic("core_models.py.tmpl"),
		"apps.py":     renderStatic("core_apps.py.tmpl"),
	}
	if err := e.writeFiles(coreApp, coreFiles); err != nil {
		return err
	}

	se, err := e.settings()
	if err != nil {
		return err
	}
	if err := se.AddToList(installedApps, coreApp); err != nil {
		return fmt.Errorf("register the core app: %w", err)
	}
	if err := se.AddToList(installedApps, accountsApp); err != nil {
		return fmt.Errorf("register the accounts app: %w", err)
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
