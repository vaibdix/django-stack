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

	if err := e.installPackages(projectPath, "Django REST Framework & drf-spectacular", PkgRestFramework, PkgDRFSpectacular); err != nil {
		return err
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

		if err := se.AddToList(InstalledApps, AppDRFSpectacular); err != nil {
			return fmt.Errorf("failed to update %s: %w", SettingsFile, err)
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
`, e.env().debug())
		se.AppendBlock(restSettings)

		spectacularSettings := fmt.Sprintf(`
SPECTACULAR_SETTINGS = {
    'TITLE': '%s API',
    'DESCRIPTION': 'API documentation for %s',
    'VERSION': '1.0.0',
    'SERVE_INCLUDE_SCHEMA': False,
}
`, e.Cfg.ProjectName, e.Cfg.ProjectName)
		se.AppendBlock(spectacularSettings)

		if err := se.Save(); err != nil {
			return err
		}
		e.Step("Added REST Framework and drf-spectacular to INSTALLED_APPS and configured settings.")
	}

	if e.Cfg.AppName != "" {
		if err := e.writeAPIPackage(projectPath, generateAPIPackage(e.Cfg.AppName, e.Cfg.SetupPydanticConf)); err != nil {
			return err
		}
	}

	endpointsMd := e.generateAPIEndpointsMd()
	endpointsPath := filepath.Join(projectPath, "API_ENDPOINTS.md")
	if err := os.WriteFile(endpointsPath, []byte(endpointsMd), 0644); err != nil {
		return fmt.Errorf("failed to create API_ENDPOINTS.md: %w", err)
	}
	e.Step("Created API_ENDPOINTS.md with API documentation URLs.")

	return nil
}

func (e *Env) generateAPIEndpointsMd() string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "# %s API Endpoints\n\n", e.Cfg.ProjectName)
	sb.WriteString("## API Documentation\n\n")
	sb.WriteString("| URL | Description |\n")
	sb.WriteString("|-----|-------------|\n")
	sb.WriteString("| [http://localhost:8000/api/schema/swagger-ui/](http://localhost:8000/api/schema/swagger-ui/) | Swagger UI (interactive API docs) |\n")
	sb.WriteString("| [http://localhost:8000/api/schema/redoc/](http://localhost:8000/api/schema/redoc/) | ReDoc (alternative API docs) |\n")
	sb.WriteString("| [http://localhost:8000/api/schema/](http://localhost:8000/api/schema/) | Raw OpenAPI schema (JSON/YAML) |\n")
	sb.WriteString("\n## Authentication\n\n")
	sb.WriteString("The API is **read-only for anonymous callers**. `GET` works without signing in; ")
	sb.WriteString("`POST`, `PUT`, `PATCH` and `DELETE` require an authenticated user, so the browsable ")
	sb.WriteString("API shows no create or edit forms until you log in.\n\n")
	sb.WriteString("Create a superuser (see README.md), then sign in at the URL below and the ")
	sb.WriteString("forms appear on every endpoint.\n\n")
	sb.WriteString("| URL | Description |\n")
	sb.WriteString("|-----|-------------|\n")
	sb.WriteString("| [http://localhost:8000/api-auth/login/](http://localhost:8000/api-auth/login/) | Log in to the browsable API |\n")
	sb.WriteString("| [http://localhost:8000/api-auth/logout/](http://localhost:8000/api-auth/logout/) | Log out |\n\n")
	sb.WriteString("To make the demo API writable without signing in, change ")
	sb.WriteString("`DEFAULT_PERMISSION_CLASSES` in settings to `rest_framework.permissions.AllowAny`.\n")

	if e.Cfg.AppName != "" {
		fmt.Fprintf(&sb, "\n## %s API\n\n", e.Cfg.AppName)
		sb.WriteString("| URL | Description |\n")
		sb.WriteString("|-----|-------------|\n")
		fmt.Fprintf(&sb, "| [http://localhost:8000/api/v1/](http://localhost:8000/api/v1/) | %s API endpoints |\n", e.Cfg.AppName)
	}

	sb.WriteString("\n## Admin\n\n")
	sb.WriteString("| URL | Description |\n")
	sb.WriteString("|-----|-------------|\n")
	sb.WriteString("| [http://localhost:8000/admin/](http://localhost:8000/admin/) | Django admin interface |\n")

	return sb.String()
}

func (e *Env) writeBookApp(projectPath string) error {
	cfg := AppTemplateConfig{
		AppName:  e.Cfg.AppName,
		Tailwind: e.Cfg.SetupTailwind,
		HTMX:     e.Cfg.SetupHTMX,
	}

	appDir := filepath.Join(projectPath, e.Cfg.AppName)

	pydantic := e.Cfg.SetupPydanticConf
	pyFiles := map[string]string{
		"models.py": generateBookModel(e.Cfg.AppName, e.hasBookPages(), e.Cfg.DjangoVersion, e.Cfg.DatabaseType),
		"admin.py":  generateAdminPy(e.Cfg.SetupMypy),
	}
	if pydantic {
		pyFiles["schemas.py"] = generateBookSchemas()
	}
	if e.Cfg.SetupHTMX {
		pyFiles["htmx.py"] = generateHtmxSupport()
	}
	if e.hasBookPages() {
		pyFiles[URLsFile] = generateAppURLs(e.Cfg.AppName)
		pyFiles["forms.py"] = generateBookForms(e.Cfg.SetupTailwind, e.Cfg.SetupMypy)
		pyFiles["views.py"] = generateBookViews(e.Cfg.AppName, e.Cfg.SetupHTMX, pydantic, e.Cfg.SetupMypy)
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
	sampleData := generateSampleData(e.Cfg.AppName, e.Cfg.SetupPydanticConf, e.Cfg.DatabaseType)
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
