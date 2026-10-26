package scaffold

import (
	"context"
	"path/filepath"

	"github.com/vaibdix/django-stack/internal/config"
)

const (
	accountsApp = "accounts"
	coreApp     = "core"
)

func (e *Env) createAccountsApp(ctx context.Context) error {
	if err := e.emit(e.sharedAppFiles()); err != nil {
		return err
	}

	e.Step("Created the core and accounts apps; the custom User model lands before the first migration.")
	return nil
}

func generateAccountsAdmin(data typingData) string {
	return render("accounts_admin.py.tmpl", data)
}

func (e *Env) sharedAppFiles() fileSet {
	files := map[string]string{
		"__init__.py": "",
		filepath.Join("migrations", "__init__.py"):     "",
		filepath.Join("migrations", "0001_initial.py"): e.accountsMigration(),
		"models.py": render("accounts_models.py.tmpl", struct{ Stubs52 bool }{e.typedAgainstStubs52()}),
		"apps.py":   renderStatic("accounts_apps.py.tmpl"),
		"admin.py":  generateAccountsAdmin(typingData{RuntimeGenerics: e.cfg.Has(config.Mypy)}),
		"tests.py": render("accounts_tests.py.tmpl", struct{ AllAuthPages, Ruff bool }{
			e.cfg.Has(config.AllAuth) && e.cfg.CreateTemplates(), e.cfg.Has(config.Ruff),
		}),
	}
	set := fileSet{}
	set.addAll(accountsApp, files)

	coreFiles := map[string]string{
		"__init__.py": "",
		"models.py":   renderStatic("core_models.py.tmpl"),
		"apps.py":     renderStatic("core_apps.py.tmpl"),
	}
	set.addAll(coreApp, coreFiles)
	return set
}
