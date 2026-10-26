package scaffold

import (
	"path/filepath"

	"github.com/vaibdix/django-stack/internal/config"
)

func (e *Env) bookFiles() fileSet {
	cfg := appTemplateConfig{
		AppName:  e.cfg.App,
		Tailwind: e.cfg.UsesTailwind(),
		HTMX:     e.cfg.UsesHTMX(),
	}
	appDir := e.cfg.App
	files := fileSet{}

	pyFiles := map[string]string{
		"fields.py": generateBookFields(typingData{RuntimeGenerics: e.cfg.Has(config.Mypy), Lazy: e.lazyAnnotations()}),
		"models.py": render("models.py.tmpl", bookModelData{AppName: e.cfg.App, HTMLViews: e.hasBookPages(), Lazy: e.lazyAnnotations()}),
		"admin.py":  generateAdminPy(typingData{RuntimeGenerics: e.cfg.Has(config.Mypy)}),
	}
	if e.cfg.UsesHTMX() {
		pyFiles["htmx.py"] = htmxSupport(e.lazyAnnotations())
	}
	if e.hasBookPages() {
		pyFiles[urlsFile] = generateAppURLs(e.cfg.App)
		pyFiles["forms.py"] = generateBookForms(bookFormsData{Tailwind: e.cfg.UsesTailwind(), RuntimeGenerics: e.cfg.Has(config.Mypy)})
		pyFiles["views.py"] = renderBookViews(bookViewsData{
			AppName:         e.cfg.App,
			HTMX:            e.cfg.UsesHTMX(),
			RuntimeGenerics: e.cfg.Has(config.Mypy),
			Stubs52:         e.typedAgainstStubs52(),
			Lazy:            e.lazyAnnotations(),
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
