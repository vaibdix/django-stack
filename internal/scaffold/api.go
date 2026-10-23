package scaffold

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vaibdix/django-stack/internal/config"
)

func (e *Env) applyRESTFramework(ctx context.Context) error {
	settingsPath := filepath.Join(e.dir, e.cfg.ModuleName(), settingsFile)
	se, err := newSettingsEditor(settingsPath)
	if err != nil {
		return err
	}

	if err := se.AddToList(installedApps, appRestFramework); err != nil {
		return fmt.Errorf("failed to update %s: %w", settingsFile, err)
	}

	for _, app := range []string{appDRFSpectacular, appDRFSpectacularSidecar} {
		if err := se.AddToList(installedApps, app); err != nil {
			return fmt.Errorf("failed to update %s: %w", settingsFile, err)
		}
	}

	restSettings := `
_DEFAULT_RENDERERS = ['rest_framework.renderers.JSONRenderer']

if DEBUG:
    _DEFAULT_RENDERERS.append('rest_framework.renderers.BrowsableAPIRenderer')

REST_FRAMEWORK = {
    'DEFAULT_PERMISSION_CLASSES': [
        'rest_framework.permissions.IsAuthenticatedOrReadOnly',
    ],
    'DEFAULT_RENDERER_CLASSES': _DEFAULT_RENDERERS,
    'DEFAULT_THROTTLE_CLASSES': [
        'rest_framework.throttling.AnonRateThrottle',
        'rest_framework.throttling.UserRateThrottle',
    ],
    'DEFAULT_THROTTLE_RATES': {
        'anon': '100/hour',
        'user': '1000/hour',
    },
    'DEFAULT_SCHEMA_CLASS': 'drf_spectacular.openapi.AutoSchema',
    'DEFAULT_PAGINATION_CLASS': 'rest_framework.pagination.PageNumberPagination',
    'PAGE_SIZE': 20,
}
`
	se.AppendBlock(restSettings)

	spectacularSettings := fmt.Sprintf(`
SPECTACULAR_SETTINGS = {
    'TITLE': '%s API',
    'DESCRIPTION': 'API documentation for %s',
    'VERSION': '1.0.0',
    'SERVE_INCLUDE_SCHEMA': False,
    'SWAGGER_UI_DIST': 'SIDECAR',
    'SWAGGER_UI_FAVICON_HREF': 'SIDECAR',
}
`, e.cfg.Name, e.cfg.Name)
	se.AppendBlock(spectacularSettings)

	if err := se.Save(); err != nil {
		return err
	}
	e.Step("Added REST Framework and drf-spectacular to INSTALLED_APPS and configured settings.")

	if e.cfg.App != "" {
		if err := e.writeAPIPackage(e.dir, generateAPIPackage(e.cfg.App)); err != nil {
			return err
		}
	}

	return nil
}

func (e *Env) writeBookApp(projectPath string) error {
	cfg := appTemplateConfig{
		AppName:  e.cfg.App,
		Tailwind: e.cfg.UsesTailwind(),
		HTMX:     e.cfg.UsesHTMX(),
	}

	appDir := filepath.Join(projectPath, e.cfg.App)

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
	for name, content := range pyFiles {
		if err := os.WriteFile(filepath.Join(appDir, name), []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to create %s: %w", name, err)
		}
		label := strings.TrimSuffix(name, ".py")
		e.Step(fmt.Sprintf("Created %s.", label))
	}

	if !e.hasBookPages() {
		return e.createSampleDataCommand(appDir)
	}

	appTemplatesDir := filepath.Join(appDir, templatesDir, e.cfg.App)
	if err := os.MkdirAll(appTemplatesDir, 0755); err != nil {
		return fmt.Errorf("failed to create app templates directory: %w", err)
	}

	htmlFiles := map[string]string{
		"book_list.html":   cfg.render("book_list"),
		"book_form.html":   cfg.render("book_form"),
		"book_delete.html": cfg.render("book_delete"),
	}
	for name, content := range htmlFiles {
		if err := os.WriteFile(filepath.Join(appTemplatesDir, name), []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to create %s: %w", name, err)
		}
		e.Step(fmt.Sprintf("Created %s template.", name))
	}

	partialsDir := filepath.Join(appTemplatesDir, "partials")
	if err := os.MkdirAll(partialsDir, 0755); err != nil {
		return fmt.Errorf("failed to create partials directory: %w", err)
	}

	partials := map[string]string{
		"book_list_items.html": cfg.render("book_list_items"),
		"book_fields.html":     cfg.render("book_fields"),
	}
	if e.cfg.UsesHTMX() {
		partials["book_form_partial.html"] = cfg.render("book_form_partial")
	}
	for name, content := range partials {
		if err := os.WriteFile(filepath.Join(partialsDir, name), []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to create %s: %w", name, err)
		}
	}
	e.Step("Created shared list and form partials.")

	return e.createSampleDataCommand(appDir)
}

func (e *Env) writeAPIPackage(projectPath string, apiFiles map[string]string) error {
	apiDir := filepath.Join(projectPath, e.cfg.App, "api")
	if err := os.MkdirAll(apiDir, 0755); err != nil {
		return fmt.Errorf("failed to create api package: %w", err)
	}
	for name, content := range apiFiles {
		if err := os.WriteFile(filepath.Join(apiDir, name), []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to create api/%s: %w", name, err)
		}
	}
	e.Step("Created api/ package.")
	return nil
}

func (e *Env) createSampleDataCommand(appDir string) error {
	managementDir := filepath.Join(appDir, "management", "commands")
	if err := os.MkdirAll(managementDir, 0755); err != nil {
		return fmt.Errorf("failed to create management directory: %w", err)
	}
	sampleData := generateSampleData(e.cfg.App, e.cfg.Database)
	if err := os.WriteFile(filepath.Join(managementDir, "create_sample_data.py"), []byte(sampleData), 0644); err != nil {
		return fmt.Errorf("failed to create sample data command: %w", err)
	}
	e.Step("Created management command for sample data.")

	for _, initDir := range []string{
		filepath.Join(appDir, "management"),
		managementDir,
	} {
		if err := os.WriteFile(filepath.Join(initDir, "__init__.py"), []byte(""), 0644); err != nil {
			return fmt.Errorf("failed to create __init__.py in %s: %w", initDir, err)
		}
	}

	return nil
}
