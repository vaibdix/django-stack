package scaffold

import (
	"fmt"
	"maps"
	"path/filepath"
	"slices"
	"strings"

	"github.com/vaibdix/django-stack/internal/config"
)

func (e *Env) writeBookApp() error {
	cfg := appTemplateConfig{
		AppName:  e.cfg.App,
		Tailwind: e.cfg.UsesTailwind(),
		HTMX:     e.cfg.UsesHTMX(),
	}

	appDir := e.cfg.App

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
	for _, name := range slices.Sorted(maps.Keys(pyFiles)) {
		if err := e.write(filepath.Join(appDir, name), []byte(pyFiles[name]), fileMode); err != nil {
			return err
		}
		e.Step(fmt.Sprintf("Created %s.", strings.TrimSuffix(name, ".py")))
	}

	if !e.hasBookPages() {
		return e.createSampleDataCommand()
	}

	appTemplatesDir := filepath.Join(appDir, templatesDir, e.cfg.App)

	htmlFiles := map[string]string{
		"book_list.html":   cfg.render("book_list"),
		"book_form.html":   cfg.render("book_form"),
		"book_delete.html": cfg.render("book_delete"),
	}
	for _, name := range slices.Sorted(maps.Keys(htmlFiles)) {
		if err := e.write(filepath.Join(appTemplatesDir, name), []byte(htmlFiles[name]), fileMode); err != nil {
			return err
		}
		e.Step(fmt.Sprintf("Created %s template.", name))
	}

	partials := map[string]string{
		"book_list_items.html": cfg.render("book_list_items"),
		"book_fields.html":     cfg.render("book_fields"),
	}
	if e.cfg.UsesHTMX() {
		partials["book_form_partial.html"] = cfg.render("book_form_partial")
	}
	if err := e.writeFiles(filepath.Join(appTemplatesDir, "partials"), partials); err != nil {
		return err
	}
	e.Step("Created shared list and form partials.")

	return e.createSampleDataCommand()
}

func (e *Env) writeAPIPackage(apiFiles map[string]string) error {
	if err := e.writeFiles(filepath.Join(e.cfg.App, "api"), apiFiles); err != nil {
		return err
	}
	e.Step("Created api/ package.")
	return nil
}

func (e *Env) createSampleDataCommand() error {
	management := filepath.Join(e.cfg.App, "management")
	if err := e.write(filepath.Join(management, "commands", "create_sample_data.py"), []byte(generateSampleData(e.cfg.App, e.cfg.Database)), fileMode); err != nil {
		return err
	}
	e.Step("Created management command for sample data.")

	return e.writeFiles(management, map[string]string{
		"__init__.py":                            "",
		filepath.Join("commands", "__init__.py"): "",
	})
}
