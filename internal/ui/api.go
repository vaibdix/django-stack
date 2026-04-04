package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func (m *Model) SetupDjangoRestFramework(projectPath string) error {
	if !m.SetupRestFramework {
		return nil
	}

	m.UpdateProgress("Setting up Django REST Framework...")

	if err := m.installAndReport(projectPath, "Django REST Framework & drf-spectacular", PkgRestFramework, PkgDRFSpectacular); err != nil {
		return err
	}

	settingsPath := filepath.Join(projectPath, m.ProjectModuleName(), SettingsFile)
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

		restSettings := `
REST_FRAMEWORK = {
    'DEFAULT_PERMISSION_CLASSES': [
        'rest_framework.permissions.AllowAny',
    ],
    'DEFAULT_RENDERER_CLASSES': [
        'rest_framework.renderers.JSONRenderer',
        'rest_framework.renderers.BrowsableAPIRenderer',
    ],
    'DEFAULT_SCHEMA_CLASS': 'drf_spectacular.openapi.AutoSchema',
    'DEFAULT_PAGINATION_CLASS': 'rest_framework.pagination.PageNumberPagination',
    'PAGE_SIZE': 20
}
`
		se.AppendBlock(restSettings)

		spectacularSettings := fmt.Sprintf(`
SPECTACULAR_SETTINGS = {
    'TITLE': '%s API',
    'DESCRIPTION': 'API documentation for %s',
    'VERSION': '1.0.0',
    'SERVE_INCLUDE_SCHEMA': False,
}
`, m.ProjectName, m.ProjectName)
		se.AppendBlock(spectacularSettings)

		if err := se.Save(); err != nil {
			return err
		}
		m.AppendStepMessage("Added REST Framework and drf-spectacular to INSTALLED_APPS and configured settings.")
	}

	if m.AppName != "" {
		if err := m.createExampleAPI(projectPath); err != nil {
			return err
		}
	}

	// Generate API_ENDPOINTS.md
	endpointsMd := m.generateAPIEndpointsMd()
	endpointsPath := filepath.Join(projectPath, "API_ENDPOINTS.md")
	if err := os.WriteFile(endpointsPath, []byte(endpointsMd), 0644); err != nil {
		return fmt.Errorf("failed to create API_ENDPOINTS.md: %w", err)
	}
	m.AppendStepMessage("Created API_ENDPOINTS.md with API documentation URLs.")

	return nil
}

func (m *Model) generateAPIEndpointsMd() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# %s API Endpoints\n\n", m.ProjectName))
	sb.WriteString("## API Documentation\n\n")
	sb.WriteString("| URL | Description |\n")
	sb.WriteString("|-----|-------------|\n")
	sb.WriteString("| [http://localhost:8000/api/schema/swagger-ui/](http://localhost:8000/api/schema/swagger-ui/) | Swagger UI (interactive API docs) |\n")
	sb.WriteString("| [http://localhost:8000/api/schema/redoc/](http://localhost:8000/api/schema/redoc/) | ReDoc (alternative API docs) |\n")
	sb.WriteString("| [http://localhost:8000/api/schema/](http://localhost:8000/api/schema/) | Raw OpenAPI schema (JSON/YAML) |\n")
	sb.WriteString("\n## Authentication\n\n")
	sb.WriteString("| URL | Description |\n")
	sb.WriteString("|-----|-------------|\n")
	sb.WriteString("| [http://localhost:8000/api-auth/](http://localhost:8000/api-auth/) | DRF browsable API login/logout |\n")

	if m.AppName != "" {
		sb.WriteString(fmt.Sprintf("\n## %s API\n\n", m.AppName))
		sb.WriteString("| URL | Description |\n")
		sb.WriteString("|-----|-------------|\n")
		sb.WriteString(fmt.Sprintf("| [http://localhost:8000/api/v1/](http://localhost:8000/api/v1/) | %s API endpoints |\n", m.AppName))
	}

	sb.WriteString("\n## Admin\n\n")
	sb.WriteString("| URL | Description |\n")
	sb.WriteString("|-----|-------------|\n")
	sb.WriteString("| [http://localhost:8000/admin/](http://localhost:8000/admin/) | Django admin interface |\n")

	return sb.String()
}

func (m *Model) createExampleAPI(projectPath string) error {
	cfg := AppTemplateConfig{
		AppName:  m.AppName,
		Tailwind: m.SetupTailwind,
		HTMX:     m.SetupHTMX,
	}

	appDir := filepath.Join(projectPath, m.AppName)

	// Write Python source files
	pyFiles := map[string]string{
		URLsFile:         generateAppURLs(m.AppName, m.SetupHTMX),
		"serializers.py": generateBookSerializer(m.AppName),
		"forms.py":       generateBookForms(m.AppName, m.SetupTailwind),
		"api.py":         generateBookAPI(m.AppName),
		"models.py":      generateBookModel(m.AppName),
		"views.py":       generateBookViews(m.AppName, m.SetupHTMX),
		"admin.py":       generateAdminPy(m.AppName),
	}
	for name, content := range pyFiles {
		if err := os.WriteFile(filepath.Join(appDir, name), []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to create %s: %w", name, err)
		}
		label := strings.TrimSuffix(name, ".py")
		m.AppendStepMessage(fmt.Sprintf("Created %s.", label))
	}

	// Write HTML templates
	appTemplatesDir := filepath.Join(appDir, TemplatesDir, m.AppName)
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
		m.AppendStepMessage(fmt.Sprintf("Created %s template.", name))
	}

	// HTMX partials
	if m.SetupHTMX {
		partialsDir := filepath.Join(appTemplatesDir, "partials")
		if err := os.MkdirAll(partialsDir, 0755); err != nil {
			return fmt.Errorf("failed to create partials directory: %w", err)
		}

		partials := map[string]string{
			"book_list_items.html":  generateBookListItemsPartial(cfg),
			"book_form_partial.html": generateBookFormPartial(cfg),
		}
		for name, content := range partials {
			if err := os.WriteFile(filepath.Join(partialsDir, name), []byte(content), 0644); err != nil {
				return fmt.Errorf("failed to create %s: %w", name, err)
			}
		}
		m.AppendStepMessage("Created HTMX partial templates.")
	}

	// Management command for sample data
	managementDir := filepath.Join(appDir, "management", "commands")
	if err := os.MkdirAll(managementDir, 0755); err != nil {
		return fmt.Errorf("failed to create management directory: %w", err)
	}
	if err := os.WriteFile(filepath.Join(managementDir, "create_sample_data.py"), []byte(generateSampleData(m.AppName)), 0644); err != nil {
		return fmt.Errorf("failed to create sample data command: %w", err)
	}
	m.AppendStepMessage("Created management command for sample data.")

	// __init__.py files for management package
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
