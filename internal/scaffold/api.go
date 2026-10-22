package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func (e *Env) applyRESTFramework(projectPath string) error {
	if !e.Cfg.SetupRestFramework {
		return nil
	}

	settingsPath := filepath.Join(projectPath, e.Cfg.ModuleName(), SettingsFile)
	se, err := NewSettingsEditor(settingsPath)
	if err != nil {
		return err
	}

	if !se.Contains("'" + AppRestFramework + "',") {
		if err := se.AddToList(InstalledApps, AppRestFramework); err != nil {
			return fmt.Errorf("failed to update %s: %w", SettingsFile, err)
		}

		for _, app := range []string{AppDRFSpectacular, AppDRFSpectacularSidecar} {
			if err := se.AddToList(InstalledApps, app); err != nil {
				return fmt.Errorf("failed to update %s: %w", SettingsFile, err)
			}
		}

		restSettings := fmt.Sprintf(`
_DEFAULT_RENDERERS = ['rest_framework.renderers.JSONRenderer']

if %s:
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
`, "DEBUG")
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
`, e.Cfg.ProjectName, e.Cfg.ProjectName)
		se.AppendBlock(spectacularSettings)

		if err := se.Save(); err != nil {
			return err
		}
		e.Step("Added REST Framework and drf-spectacular to INSTALLED_APPS and configured settings.")
	}

	if e.Cfg.AppName != "" {
		if err := e.writeAPIPackage(projectPath, generateAPIPackage(e.Cfg.AppName)); err != nil {
			return err
		}
	}

	return nil
}

func (e *Env) writeBookApp(projectPath string) error {
	cfg := AppTemplateConfig{
		AppName:  e.Cfg.AppName,
		Tailwind: e.Cfg.SetupTailwind,
		HTMX:     e.Cfg.SetupHTMX,
	}

	appDir := filepath.Join(projectPath, e.Cfg.AppName)

	pyFiles := map[string]string{
		"fields.py": generateBookFields(e.Cfg.SetupMypy),
		"models.py": generateBookModel(e.Cfg.AppName, e.hasBookPages(), e.Cfg.DjangoVersion, e.Cfg.DatabaseType),
		"admin.py":  generateAdminPy(e.Cfg.SetupMypy),
	}
	if e.Cfg.SetupHTMX {
		pyFiles["htmx.py"] = generateHtmxSupport()
	}
	if e.hasBookPages() {
		pyFiles[URLsFile] = generateAppURLs(e.Cfg.AppName)
		pyFiles["forms.py"] = generateBookForms(e.Cfg.SetupTailwind, e.Cfg.SetupMypy)
		pyFiles["views.py"] = renderBookViews(bookViewsData{
			AppName:         e.Cfg.AppName,
			HTMX:            e.Cfg.SetupHTMX,
			RuntimeGenerics: e.Cfg.SetupMypy,
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

	appTemplatesDir := filepath.Join(appDir, TemplatesDir, e.Cfg.AppName)
	if err := os.MkdirAll(appTemplatesDir, 0755); err != nil {
		return fmt.Errorf("failed to create app templates directory: %w", err)
	}

	htmlFiles := map[string]string{
		"book_list.html":   generateBookListTemplate(cfg),
		"book_form.html":   generateBookFormTemplate(cfg),
		"book_delete.html": generateBookDeleteTemplate(cfg),
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
		"book_list_items.html": generateBookListItemsPartial(cfg),
		"book_fields.html":     generateBookFieldsPartial(cfg),
	}
	if e.Cfg.SetupHTMX {
		partials["book_form_partial.html"] = generateBookFormPartial(cfg)
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
	apiDir := filepath.Join(projectPath, e.Cfg.AppName, "api")
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
	sampleData := generateSampleData(e.Cfg.AppName, e.Cfg.DatabaseType)
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
