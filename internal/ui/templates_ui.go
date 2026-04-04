package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func (m *Model) SetupGlobalTemplates(projectPath string) error {
	m.UseGlobalTemplates = true

	globalTemplatesPath := filepath.Join(projectPath, "templates")
	if err := os.MkdirAll(globalTemplatesPath, 0755); err != nil {
		return fmt.Errorf("failed to create global templates directory: %w", err)
	}
	staticPath := filepath.Join(projectPath, "static")
	if err := os.MkdirAll(filepath.Join(staticPath, "css"), 0755); err != nil {
		return fmt.Errorf("failed to create static/css directory: %w", err)
	}
	if err := os.MkdirAll(filepath.Join(staticPath, "js"), 0755); err != nil {
		return fmt.Errorf("failed to create static/js directory: %w", err)
	}
	if err := os.WriteFile(filepath.Join(staticPath, "js", "main.js"), []byte("// Main JavaScript\n"), 0644); err != nil {
		return fmt.Errorf("failed to create main.js: %w", err)
	}

	config := map[string]string{
		"tailwind":         boolToString(m.SetupTailwind),
		"api":              boolToString(m.SetupRestFramework),
		"global_templates": boolToString(m.UseGlobalTemplates),
		"app_name":         m.AppName,
		"htmx":             boolToString(m.SetupHTMX),
		"allauth":          boolToString(m.SetupAllAuth),
	}

	baseContent := GetTemplateContent("base.html", config)
	if err := os.WriteFile(filepath.Join(globalTemplatesPath, "base.html"), []byte(baseContent), 0644); err != nil {
		return fmt.Errorf("failed to create base.html: %w", err)
	}

	indexContent := GetTemplateContent("index.html", config)
	if err := os.WriteFile(filepath.Join(globalTemplatesPath, "index.html"), []byte(indexContent), 0644); err != nil {
		return fmt.Errorf("failed to create index.html: %w", err)
	}

	if !m.SetupTailwind && m.UseGlobalTemplates {
		if err := os.WriteFile(filepath.Join(staticPath, "css", "base.css"), []byte(GetBaseCSS()), 0644); err != nil {
			return fmt.Errorf("failed to create base.css: %w", err)
		}

		if err := os.WriteFile(filepath.Join(staticPath, "css", "home.css"), []byte(GetHomeCSS()), 0644); err != nil {
			return fmt.Errorf("failed to create home.css: %w", err)
		}
	}

	m.AppendStepMessage("Created global templates and static files.")
	m.UpdateProgress("Setting up global templates and static files...")
	return nil
}

func (m *Model) SetupProjectUrls(projectPath string) error {
	projectModulePath := filepath.Join(projectPath, m.ProjectModuleName())
	contextProcessorsPath := filepath.Join(projectModulePath, "context_processors.py")
	appNameValue := "None"
	if m.AppName != "" {
		appNameValue = "'" + m.AppName + "'"
	}
	contextProcessorsContent := fmt.Sprintf(`def project_context(request):
    return {
        'project_name': '%s',
        'app_name': %s
    }
`, m.ProjectName, appNameValue)

	if err := os.WriteFile(contextProcessorsPath, []byte(contextProcessorsContent), 0644); err != nil {
		return fmt.Errorf("failed to create context_processors.py: %w", err)
	}

	viewsContent := m.buildProjectViewsContent()
	if viewsContent != "" {
		viewsPath := filepath.Join(projectModulePath, "views.py")
		if err := os.WriteFile(viewsPath, []byte(viewsContent), 0644); err != nil {
			return fmt.Errorf("failed to create views.py: %w", err)
		}
	}

	urlsContent := m.buildProjectURLsContent()
	urlsPath := filepath.Join(projectModulePath, URLsFile)
	if err := os.WriteFile(urlsPath, []byte(urlsContent), 0644); err != nil {
		return fmt.Errorf("failed to create urls.py: %w", err)
	}

	return nil
}

func (m *Model) buildProjectViewsContent() string {
	if !m.CreateTemplates {
		return ""
	}

	sections := []string{
		"from django.views.generic import TemplateView",
		`class HomeView(TemplateView):
    template_name = 'index.html'`,
	}

	return strings.Join(sections, "\n\n")
}

func (m *Model) buildProjectURLsContent() string {
	imports := []string{"path"}
	if m.needsIncludeInProjectURLs() {
		imports = append(imports, "include")
	}

	lines := []string{
		"from django.contrib import admin",
		fmt.Sprintf("from django.urls import %s", strings.Join(imports, ", ")),
	}

	if m.SetupRestFramework {
		lines = append(lines, "from drf_spectacular.views import SpectacularAPIView, SpectacularSwaggerView, SpectacularRedocView")
	}

	if m.CreateTemplates {
		lines = append(lines, "")
		lines = append(lines, "from . import views")
	}

	lines = append(lines, "", "urlpatterns = [")

	if m.CreateTemplates {
		lines = append(lines, "    path('', views.HomeView.as_view(), name='home'),")
	}

	lines = append(lines, "    path('admin/', admin.site.urls),")
	lines = append(lines, "    path('__reload__/', include('django_browser_reload.urls')),")

	if m.SetupAllAuth {
		lines = append(lines, "    path('accounts/', include('allauth.urls')),")
	}

	if m.SetupRestFramework {
		lines = append(lines, "    # drf-spectacular API documentation")
		lines = append(lines, "    path('api/schema/', SpectacularAPIView.as_view(), name='schema'),")
		lines = append(lines, "    path('api/schema/swagger-ui/', SpectacularSwaggerView.as_view(url_name='schema'), name='swagger-ui'),")
		lines = append(lines, "    path('api/schema/redoc/', SpectacularRedocView.as_view(url_name='schema'), name='redoc'),")
		lines = append(lines, "    path('api-auth/', include('rest_framework.urls', namespace='rest_framework')),")
		if m.AppName != "" {
			lines = append(lines, fmt.Sprintf("    path('api/v1/', include('%s.api')),", m.AppName))
			lines = append(lines, fmt.Sprintf("    path('%s/', include('%s.urls')),", m.AppName, m.AppName))
		}
	} else if m.AppName != "" && m.CreateAppTemplates {
		lines = append(lines, fmt.Sprintf("    path('%s/', include('%s.urls', namespace='%s')),", m.AppName, m.AppName, m.AppName))
	}

	lines = append(lines, "]")
	return strings.Join(lines, "\n")
}

func (m *Model) needsIncludeInProjectURLs() bool {
	return m.SetupRestFramework || m.AppName != "" || true
}
