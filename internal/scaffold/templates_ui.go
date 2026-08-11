package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vaibdix/django-cli/internal/assets"
)

func (e *Env) applyGlobalTemplates(projectPath string) error {
	e.Cfg.UseGlobalTemplates = true

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

	opts := e.pageOptions()

	baseContent := assets.RenderPage(assets.PageBase, opts)
	if err := os.WriteFile(filepath.Join(globalTemplatesPath, "base.html"), []byte(baseContent), 0644); err != nil {
		return fmt.Errorf("failed to create base.html: %w", err)
	}

	indexContent := assets.RenderPage(assets.PageIndex, opts)
	if err := os.WriteFile(filepath.Join(globalTemplatesPath, "index.html"), []byte(indexContent), 0644); err != nil {
		return fmt.Errorf("failed to create index.html: %w", err)
	}

	if !e.Cfg.SetupTailwind && e.Cfg.UseGlobalTemplates {
		if err := os.WriteFile(filepath.Join(staticPath, "css", "base.css"), []byte(assets.BaseCSS()), 0644); err != nil {
			return fmt.Errorf("failed to create base.css: %w", err)
		}

		if err := os.WriteFile(filepath.Join(staticPath, "css", "home.css"), []byte(assets.HomeCSS()), 0644); err != nil {
			return fmt.Errorf("failed to create home.css: %w", err)
		}

		if err := os.WriteFile(filepath.Join(staticPath, "css", "app.css"), []byte(assets.AppCSS()), 0644); err != nil {
			return fmt.Errorf("failed to create app.css: %w", err)
		}

		if err := os.WriteFile(filepath.Join(staticPath, "css", "auth.css"), []byte(assets.AuthCSS()), 0644); err != nil {
			return fmt.Errorf("failed to create auth.css: %w", err)
		}
	}

	e.Step("Created global templates and static files.")
	return nil
}

func (e *Env) writeProjectURLs(projectPath string) error {
	projectModulePath := filepath.Join(projectPath, e.Cfg.ModuleName())
	contextProcessorsPath := filepath.Join(projectModulePath, "context_processors.py")
	appNameValue := "None"
	if e.Cfg.AppName != "" {
		appNameValue = "'" + e.Cfg.AppName + "'"
	}
	contextProcessorsContent := fmt.Sprintf(`def project_context(request):
    return {
        'project_name': '%s',
        'app_name': %s
    }
`, e.Cfg.ProjectName, appNameValue)

	if err := os.WriteFile(contextProcessorsPath, []byte(contextProcessorsContent), 0644); err != nil {
		return fmt.Errorf("failed to create context_processors.py: %w", err)
	}

	viewsContent := e.buildProjectViewsContent()
	if viewsContent != "" {
		viewsPath := filepath.Join(projectModulePath, "views.py")
		if err := os.WriteFile(viewsPath, []byte(viewsContent), 0644); err != nil {
			return fmt.Errorf("failed to create views.py: %w", err)
		}
	}

	urlsContent := e.buildProjectURLsContent()
	urlsPath := filepath.Join(projectModulePath, URLsFile)
	if err := os.WriteFile(urlsPath, []byte(urlsContent), 0644); err != nil {
		return fmt.Errorf("failed to create urls.py: %w", err)
	}

	return nil
}

func (e *Env) buildProjectViewsContent() string {
	if !e.Cfg.CreateTemplates {
		return ""
	}

	sections := []string{
		"from django.views.generic import TemplateView",
		`class HomeView(TemplateView):
    template_name = 'index.html'`,
	}

	return strings.Join(sections, "\n\n")
}

func (e *Env) apiOnlyHomeTarget() string {
	if e.Cfg.SetupRestFramework {
		return "/api/schema/swagger-ui/"
	}
	return "/admin/"
}

func (e *Env) buildProjectURLsContent() string {

	lines := []string{
		"from django.contrib import admin",
		"from django.urls import path, include",
	}

	if e.Cfg.APIOnly() {
		lines = append(lines, "from django.views.generic import RedirectView")
	}

	if e.Cfg.SetupRestFramework {
		lines = append(lines, "from drf_spectacular.views import SpectacularAPIView, SpectacularSwaggerView, SpectacularRedocView")
	}

	if e.Cfg.CreateTemplates {
		lines = append(lines, "")
		lines = append(lines, "from . import views")
	}

	lines = append(lines, "", "urlpatterns = [")

	if e.Cfg.CreateTemplates {
		lines = append(lines, "    path('', views.HomeView.as_view(), name='home'),")
	} else {
		// An API-only project still needs a root URL. Landing on a 404 at / is
		// what made the old templates-less combination look broken.
		lines = append(lines, fmt.Sprintf(
			"    path('', RedirectView.as_view(url='%s', permanent=False), name='home'),", e.apiOnlyHomeTarget()))
	}

	lines = append(lines, "    path('admin/', admin.site.urls),")
	lines = append(lines, "    path('__reload__/', include('django_browser_reload.urls')),")

	if e.Cfg.SetupAllAuth {
		lines = append(lines, "    path('accounts/', include('allauth.urls')),")
	}

	if e.Cfg.SetupRestFramework {
		lines = append(lines, "    # drf-spectacular API documentation")
		lines = append(lines, "    path('api/schema/', SpectacularAPIView.as_view(), name='schema'),")
		lines = append(lines, "    path('api/schema/swagger-ui/', SpectacularSwaggerView.as_view(url_name='schema'), name='swagger-ui'),")
		lines = append(lines, "    path('api/schema/redoc/', SpectacularRedocView.as_view(url_name='schema'), name='redoc'),")
		lines = append(lines, "    path('api-auth/', include('rest_framework.urls', namespace='rest_framework')),")
		if e.Cfg.AppName != "" {
			lines = append(lines, fmt.Sprintf("    path('api/v1/', include('%s.api')),", e.Cfg.AppName))
			if !e.Cfg.APIOnly() {
				lines = append(lines, fmt.Sprintf("    path('%s/', include('%s.urls')),", e.Cfg.AppName, e.Cfg.AppName))
			}
		}
	} else if e.Cfg.AppName != "" && e.Cfg.CreateAppTemplates {
		lines = append(lines, fmt.Sprintf("    path('%s/', include('%s.urls', namespace='%s')),", e.Cfg.AppName, e.Cfg.AppName, e.Cfg.AppName))
	}

	lines = append(lines, "]")
	return strings.Join(lines, "\n")
}
