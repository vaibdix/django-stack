package render

import (
	"path/filepath"

	"github.com/vaibdix/django-stack/internal/config"
)

const (
	accountsApp = "accounts"
	coreApp     = "core"
)

func generateAccountsAdmin(data typingData) string {
	return Render("accounts_admin.py.tmpl", data)
}

func (p Project) sharedAppFiles() FileSet {
	files := map[string]string{
		"__init__.py": "",
		filepath.Join("migrations", "__init__.py"):     "",
		filepath.Join("migrations", "0001_initial.py"): p.accountsMigration(),
		"models.py": Render("accounts_models.py.tmpl", struct{ Stubs52 bool }{p.typedAgainstStubs52()}),
		"apps.py":   RenderStatic("accounts_apps.py.tmpl"),
		"admin.py":  generateAccountsAdmin(typingData{RuntimeGenerics: p.Cfg.Has(config.Mypy)}),
		"tests.py": Render("accounts_tests.py.tmpl", struct{ AllAuthPages, Ruff bool }{
			p.Cfg.Has(config.AllAuth) && p.Cfg.CreateTemplates(), p.Cfg.Has(config.Ruff),
		}),
	}
	set := FileSet{}
	set.addAll(accountsApp, files)

	coreFiles := map[string]string{
		"__init__.py": "",
		"models.py":   RenderStatic("core_models.py.tmpl"),
		"apps.py":     RenderStatic("core_apps.py.tmpl"),
	}
	set.addAll(coreApp, coreFiles)
	return set
}
