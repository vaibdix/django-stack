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

func (p project) appNames() appNames {
	return appNames{AppName: p.cfg.App().String(), AppNameTitle: titleCase(p.cfg.App().String())}
}

func (p project) appFiles() fileSet {
	files := fileSet{}
	if !p.cfg.App().Ok() {
		return files
	}
	files.add(filepath.Join(p.cfg.App().String(), "__init__.py"), "", fileMode)
	files.add(filepath.Join(p.cfg.App().String(), "migrations", "__init__.py"), "", fileMode)
	files.add(filepath.Join(p.cfg.App().String(), "views.py"), renderStatic("app_views.py.tmpl"), fileMode)
	files.add(filepath.Join(p.cfg.App().String(), "apps.py"), render("apps.py.tmpl", p.appNames()), fileMode)
	files.addAll(p.cfg.App().String(), p.testFiles())
	maps.Copy(files, p.bookFiles())
	maps.Copy(files, p.appMigrationFiles())
	return files
}

type testsData struct {
	appNames
	HTMX bool
	Ruff bool
}

func (p project) testsData() testsData {
	return testsData{appNames: p.appNames(), HTMX: p.cfg.UsesHTMX(), Ruff: p.cfg.Has(config.Ruff)}
}

func (p project) apiTestsContent() string {
	if p.cfg.Has(config.Ninja) {
		return render("tests_ninja.py.tmpl", p.testsData())
	}
	return render("tests_api.py.tmpl", p.testsData())
}

func (p project) testFiles() map[string]string {
	files := map[string]string{
		filepath.Join("tests", "__init__.py"):    "",
		filepath.Join("tests", "test_models.py"): render("tests_model.py.tmpl", p.appNames()),
	}
	if p.hasBookPages() {
		files[filepath.Join("tests", "test_pages.py")] = render("tests_book.py.tmpl", p.testsData())
	}
	if p.cfg.HasAPI() {
		files[filepath.Join("tests", "test_api.py")] = p.apiTestsContent()
	}
	return files
}
