package scaffold

import (
	"maps"
	"path/filepath"
	"strings"

	"github.com/vaibdix/django-stack/internal/config"
)

func titleCase(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

type appNames struct {
	AppName      string
	AppNameTitle string
}

func (e *Env) appNames() appNames {
	return appNames{AppName: e.cfg.App, AppNameTitle: titleCase(e.cfg.App)}
}

func (e *Env) appFiles() fileSet {
	files := fileSet{}
	if e.cfg.App == "" {
		return files
	}
	files.add(filepath.Join(e.cfg.App, "__init__.py"), "", fileMode)
	files.add(filepath.Join(e.cfg.App, "migrations", "__init__.py"), "", fileMode)
	files.add(filepath.Join(e.cfg.App, "views.py"), renderStatic("app_views.py.tmpl"), fileMode)
	files.add(filepath.Join(e.cfg.App, "apps.py"), render("apps.py.tmpl", e.appNames()), fileMode)
	files.addAll(e.cfg.App, e.testFiles())
	maps.Copy(files, e.bookFiles())
	maps.Copy(files, e.appMigrationFiles())
	return files
}

type testsData struct {
	appNames
	HTMX bool
	Ruff bool
}

func (e *Env) testsData() testsData {
	return testsData{appNames: e.appNames(), HTMX: e.cfg.UsesHTMX(), Ruff: e.cfg.Has(config.Ruff)}
}

func (e *Env) apiTestsContent() string {
	if e.cfg.Has(config.Ninja) {
		return render("tests_ninja.py.tmpl", e.testsData())
	}
	return render("tests_api.py.tmpl", e.testsData())
}

func (e *Env) testFiles() map[string]string {
	files := map[string]string{
		filepath.Join("tests", "__init__.py"):    "",
		filepath.Join("tests", "test_models.py"): render("tests_model.py.tmpl", e.appNames()),
	}
	if e.hasBookPages() {
		files[filepath.Join("tests", "test_pages.py")] = render("tests_book.py.tmpl", e.testsData())
	}
	if e.cfg.HasAPI() {
		files[filepath.Join("tests", "test_api.py")] = e.apiTestsContent()
	}
	return files
}
