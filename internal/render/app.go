package render

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

func (p Project) appNames() appNames {
	return appNames{AppName: p.Cfg.App().String(), AppNameTitle: titleCase(p.Cfg.App().String())}
}

func (p Project) appFiles() FileSet {
	files := FileSet{}
	if !p.Cfg.App().Ok() {
		return files
	}
	files.add(filepath.Join(p.Cfg.App().String(), "__init__.py"), "", FileMode)
	files.add(filepath.Join(p.Cfg.App().String(), "migrations", "__init__.py"), "", FileMode)
	files.add(filepath.Join(p.Cfg.App().String(), "views.py"), RenderStatic("app_views.py.tmpl"), FileMode)
	files.add(filepath.Join(p.Cfg.App().String(), "apps.py"), Render("apps.py.tmpl", p.appNames()), FileMode)
	files.addAll(p.Cfg.App().String(), p.testFiles())
	maps.Copy(files, p.bookFiles())
	maps.Copy(files, p.appMigrationFiles())
	return files
}

type testsData struct {
	appNames
	HTMX bool
	Ruff bool
}

func (p Project) testsData() testsData {
	return testsData{appNames: p.appNames(), HTMX: p.Cfg.UsesHTMX(), Ruff: p.Cfg.Has(config.Ruff)}
}

func (p Project) apiTestsContent() string {
	if p.Cfg.Has(config.Ninja) {
		return Render("tests_ninja.py.tmpl", p.testsData())
	}
	return Render("tests_api.py.tmpl", p.testsData())
}

func (p Project) testFiles() map[string]string {
	files := map[string]string{
		filepath.Join("tests", "__init__.py"):    "",
		filepath.Join("tests", "test_models.py"): Render("tests_model.py.tmpl", p.appNames()),
	}
	if p.hasBookPages() {
		files[filepath.Join("tests", "test_pages.py")] = Render("tests_book.py.tmpl", p.testsData())
	}
	if p.Cfg.HasAPI() {
		files[filepath.Join("tests", "test_api.py")] = p.apiTestsContent()
	}
	return files
}
