package scaffold

import (
	"path/filepath"

	"github.com/vaibdix/django-stack/internal/config"
)

func (e *Env) writeBookApp() error {
	if err := e.emit(e.bookFiles()); err != nil {
		return err
	}
	e.Step("Created the book models, admin, sample data command and pages.")
	return nil
}

func (e *Env) bookFiles() fileSet {
	cfg := appTemplateConfig{
		AppName:  e.cfg.App,
		Tailwind: e.cfg.UsesTailwind(),
		HTMX:     e.cfg.UsesHTMX(),
	}
	appDir := e.cfg.App
	files := fileSet{}

	pyFiles := map[string]string{
		"fields.py": generateBookFields(typingData{RuntimeGenerics: e.cfg.Has(config.Mypy)}),
		"models.py": generateBookModel(e.cfg.App, e.hasBookPages()),
		"admin.py":  generateAdminPy(typingData{RuntimeGenerics: e.cfg.Has(config.Mypy)}),
	}
	if e.cfg.UsesHTMX() {
		pyFiles["htmx.py"] = generateHtmxSupport()
	}
	if e.hasBookPages() {
		pyFiles[urlsFile] = generateAppURLs(e.cfg.App)
		pyFiles["forms.py"] = generateBookForms(bookFormsData{Tailwind: e.cfg.UsesTailwind(), RuntimeGenerics: e.cfg.Has(config.Mypy)})
		pyFiles["views.py"] = renderBookViews(bookViewsData{
			AppName:         e.cfg.App,
			HTMX:            e.cfg.UsesHTMX(),
			RuntimeGenerics: e.cfg.Has(config.Mypy),
			Stubs52:         e.typedAgainstStubs52(),
		})
	}
	files.addAll(appDir, pyFiles)

	management := filepath.Join(appDir, "management")
	files.addAll(management, map[string]string{
		"__init__.py":                                      "",
		filepath.Join("commands", "__init__.py"):           "",
		filepath.Join("commands", "create_sample_data.py"): generateSampleData(e.cfg.App, e.cfg.Database),
	})

	if !e.hasBookPages() {
		return files
	}

	appTemplatesDir := filepath.Join(appDir, templatesDir, e.cfg.App)
	files.addAll(appTemplatesDir, map[string]string{
		"book_list.html":   cfg.render("book_list"),
		"book_form.html":   cfg.render("book_form"),
		"book_delete.html": cfg.render("book_delete"),
	})

	partials := map[string]string{
		"book_list_items.html": cfg.render("book_list_items"),
		"book_fields.html":     cfg.render("book_fields"),
	}
	if e.cfg.UsesHTMX() {
		partials["book_form_partial.html"] = cfg.render("book_form_partial")
	}
	files.addAll(filepath.Join(appTemplatesDir, "partials"), partials)
	return files
}

func (e *Env) apiFiles() fileSet {
	files := fileSet{}
	switch {
	case e.cfg.App == "":
	case e.cfg.Has(config.RestFramework):
		files.addAll(filepath.Join(e.cfg.App, "api"), generateAPIPackage(e.cfg.App))
	case e.cfg.Has(config.Ninja):
		files.addAll(filepath.Join(e.cfg.App, "api"), generateNinjaPackage(e.cfg.App, e.cfg.Name))
	}
	return files
}

func (e *Env) writeAPIPackage() error {
	if err := e.emit(e.apiFiles()); err != nil {
		return err
	}
	e.Step("Created api/ package.")
	return nil
}
