package scaffold

import (
	"path/filepath"

	"github.com/vaibdix/django-stack/internal/config"
)

func (p project) bookFiles() fileSet {
	cfg := appTemplateConfig{
		AppName:  p.cfg.App,
		Tailwind: p.cfg.UsesTailwind(),
		HTMX:     p.cfg.UsesHTMX(),
	}
	appDir := p.cfg.App
	files := fileSet{}

	pyFiles := map[string]string{
		"fields.py": generateBookFields(typingData{RuntimeGenerics: p.cfg.Has(config.Mypy), Lazy: p.lazyAnnotations()}),
		"models.py": render("models.py.tmpl", bookModelData{AppName: p.cfg.App, HTMLViews: p.hasBookPages(), Lazy: p.lazyAnnotations()}),
		"admin.py":  generateAdminPy(typingData{RuntimeGenerics: p.cfg.Has(config.Mypy)}),
	}
	if p.cfg.UsesHTMX() {
		pyFiles["htmx.py"] = htmxSupport(p.lazyAnnotations())
	}
	if p.hasBookPages() {
		pyFiles[urlsFile] = generateAppURLs(p.cfg.App)
		pyFiles["forms.py"] = generateBookForms(bookFormsData{Tailwind: p.cfg.UsesTailwind(), RuntimeGenerics: p.cfg.Has(config.Mypy)})
		pyFiles["views.py"] = renderBookViews(bookViewsData{
			AppName:         p.cfg.App,
			HTMX:            p.cfg.UsesHTMX(),
			RuntimeGenerics: p.cfg.Has(config.Mypy),
			Stubs52:         p.typedAgainstStubs52(),
			Lazy:            p.lazyAnnotations(),
		})
	}
	files.addAll(appDir, pyFiles)

	management := filepath.Join(appDir, "management")
	files.addAll(management, map[string]string{
		"__init__.py":                                      "",
		filepath.Join("commands", "__init__.py"):           "",
		filepath.Join("commands", "create_sample_data.py"): generateSampleData(p.cfg.App, p.cfg.Database),
	})

	if !p.hasBookPages() {
		return files
	}

	appTemplatesDir := filepath.Join(appDir, templatesDir, p.cfg.App)
	files.addAll(appTemplatesDir, map[string]string{
		"book_list.html":   cfg.render("book_list"),
		"book_form.html":   cfg.render("book_form"),
		"book_delete.html": cfg.render("book_delete"),
	})

	partials := map[string]string{
		"book_list_items.html": cfg.render("book_list_items"),
		"book_fields.html":     cfg.render("book_fields"),
	}
	if p.cfg.UsesHTMX() {
		partials["book_form_partial.html"] = cfg.render("book_form_partial")
	}
	files.addAll(filepath.Join(appTemplatesDir, "partials"), partials)
	return files
}
