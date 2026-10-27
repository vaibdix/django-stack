package render

import (
	"path/filepath"

	"github.com/vaibdix/django-stack/internal/config"
)

func (p Project) bookFiles() FileSet {
	cfg := AppTemplateConfig{
		AppName:  p.Cfg.App().String(),
		Tailwind: p.Cfg.UsesTailwind(),
		HTMX:     p.Cfg.UsesHTMX(),
	}
	appDir := p.Cfg.App().String()
	files := FileSet{}

	pyFiles := map[string]string{
		"fields.py": generateBookFields(typingData{RuntimeGenerics: p.Cfg.Has(config.Mypy), Lazy: p.lazyAnnotations()}),
		"models.py": Render("models.py.tmpl", bookModelData{AppName: p.Cfg.App().String(), HTMLViews: p.hasBookPages(), Lazy: p.lazyAnnotations()}),
		"admin.py":  generateAdminPy(typingData{RuntimeGenerics: p.Cfg.Has(config.Mypy)}),
	}
	if p.Cfg.UsesHTMX() {
		pyFiles["htmx.py"] = htmxSupport(p.lazyAnnotations())
	}
	if p.hasBookPages() {
		pyFiles[urlsFile] = generateAppURLs(p.Cfg.App().String())
		pyFiles["forms.py"] = generateBookForms(bookFormsData{Tailwind: p.Cfg.UsesTailwind(), RuntimeGenerics: p.Cfg.Has(config.Mypy)})
		pyFiles["views.py"] = renderBookViews(bookViewsData{
			AppName:         p.Cfg.App().String(),
			HTMX:            p.Cfg.UsesHTMX(),
			RuntimeGenerics: p.Cfg.Has(config.Mypy),
			Stubs52:         p.typedAgainstStubs52(),
			Lazy:            p.lazyAnnotations(),
		})
	}
	files.addAll(appDir, pyFiles)

	management := filepath.Join(appDir, "management")
	files.addAll(management, map[string]string{
		"__init__.py":                                      "",
		filepath.Join("commands", "__init__.py"):           "",
		filepath.Join("commands", "create_sample_data.py"): generateSampleData(p.Cfg.App().String(), p.Cfg.Database()),
	})

	if !p.hasBookPages() {
		return files
	}

	appTemplatesDir := filepath.Join(appDir, templatesDir, p.Cfg.App().String())
	files.addAll(appTemplatesDir, map[string]string{
		"book_list.html":   cfg.Render("book_list"),
		"book_form.html":   cfg.Render("book_form"),
		"book_delete.html": cfg.Render("book_delete"),
	})

	partials := map[string]string{
		"book_list_items.html": cfg.Render("book_list_items"),
		"book_fields.html":     cfg.Render("book_fields"),
	}
	if p.Cfg.UsesHTMX() {
		partials["book_form_partial.html"] = cfg.Render("book_form_partial")
	}
	files.addAll(filepath.Join(appTemplatesDir, "partials"), partials)
	return files
}
