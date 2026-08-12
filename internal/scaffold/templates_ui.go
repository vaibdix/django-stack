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
	if err := os.WriteFile(filepath.Join(staticPath, "js", "main.js"), []byte(e.mainJS()), 0644); err != nil {
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

	for _, page := range assets.ErrorPages {
		name := page.Code + ".html"
		content := assets.ErrorTemplate(page, e.Cfg.SetupTailwind)
		if err := os.WriteFile(filepath.Join(globalTemplatesPath, name), []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to create %s: %w", name, err)
		}
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

func (e *Env) mainJS() string {
	if !e.Cfg.SetupHTMX {
		return "export {};\n"
	}

	return `document.body.addEventListener('htmx:beforeSwap', function (event) {
    if (event.detail.xhr.status === 422) {
        event.detail.shouldSwap = true;
        event.detail.isError = false;
    }
});
`
}

func (e *Env) renderContextProcessors() string {
	appNameValue, appNameType := "None", "str | None"
	if e.Cfg.AppName != "" {
		appNameValue, appNameType = "'"+e.Cfg.AppName+"'", "str"
	}

	return fmt.Sprintf(`from typing import TypedDict

from django.http import HttpRequest

class ProjectContext(TypedDict):
    project_name: str
    app_name: %s

def project_context(request: HttpRequest) -> ProjectContext:
    return {
        'project_name': '%s',
        'app_name': %s,
    }
`, appNameType, e.Cfg.ProjectName, appNameValue)
}

func (e *Env) writeProjectURLs(projectPath string) error {
	projectModulePath := filepath.Join(projectPath, e.Cfg.ModuleName())
	contextProcessorsPath := filepath.Join(projectModulePath, "context_processors.py")
	if err := os.WriteFile(contextProcessorsPath, []byte(e.renderContextProcessors()), 0644); err != nil {
		return fmt.Errorf("failed to create context_processors.py: %w", err)
	}

	viewsPath := filepath.Join(projectModulePath, "views.py")
	if err := os.WriteFile(viewsPath, []byte(e.buildProjectViewsContent()), 0644); err != nil {
		return fmt.Errorf("failed to create views.py: %w", err)
	}

	urlsContent := e.buildProjectURLsContent()
	urlsPath := filepath.Join(projectModulePath, URLsFile)
	if err := os.WriteFile(urlsPath, []byte(urlsContent), 0644); err != nil {
		return fmt.Errorf("failed to create urls.py: %w", err)
	}

	return nil
}

func (e *Env) buildProjectViewsContent() string {
	return render("project_views.py.tmpl", struct{ Templates bool }{Templates: e.Cfg.CreateTemplates})
}

func (e *Env) apiOnlyHomeTarget() string {
	if e.Cfg.SetupRestFramework {
		return "/api/schema/swagger-ui/"
	}
	return "/admin/"
}

func (e *Env) buildProjectURLsContent() string {

	lines := []string{
		"from django.conf import settings",
		"from django.contrib import admin",
		"from django.urls import path, include",
	}

	if e.Cfg.APIOnly() {
		lines = append(lines, "from django.views.generic import RedirectView")
	}

	if e.Cfg.SetupRestFramework {
		lines = append(lines, "from drf_spectacular.views import SpectacularAPIView, SpectacularSwaggerView, SpectacularRedocView")
	}

	lines = append(lines, "")
	lines = append(lines, "from . import views")

	lines = append(lines, "", "urlpatterns = [")

	if e.Cfg.CreateTemplates {
		lines = append(lines, "    path('', views.HomeView.as_view(), name='home'),")
	} else {
		lines = append(lines, fmt.Sprintf(
			"    path('', RedirectView.as_view(url='%s', permanent=False), name='home'),", e.apiOnlyHomeTarget()))
	}

	lines = append(lines, "    path(settings.ADMIN_URL, admin.site.urls),")
	lines = append(lines, "    path('healthz/', views.healthz, name='healthz'),")

	if e.Cfg.SetupAllAuth {
		lines = append(lines, "    path('accounts/', include('allauth.urls')),")
	}

	if e.Cfg.SetupRestFramework {
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
	lines = append(lines, "")
	lines = append(lines, "handler400 = 'MODULE.views.bad_request'")
	lines = append(lines, "handler403 = 'MODULE.views.permission_denied'")
	lines = append(lines, "handler404 = 'MODULE.views.page_not_found'")
	lines = append(lines, "handler500 = 'MODULE.views.server_error'")
	lines = append(lines, "")
	lines = append(lines, "if settings.DEBUG:")
	lines = append(lines, "    urlpatterns += [path('__reload__/', include('django_browser_reload.urls'))]")

	return strings.ReplaceAll(strings.Join(lines, "\n"), "MODULE.views", e.Cfg.ModuleName()+".views")
}
