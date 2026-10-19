package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vaibdix/django-cli/internal/config"

	"github.com/vaibdix/django-cli/internal/assets"
)

func (e *Env) applyGlobalTemplates(projectPath string) error {
	e.Cfg.UseGlobalTemplates = true

	globalTemplatesPath := filepath.Join(projectPath, "templates")
	if err := os.MkdirAll(globalTemplatesPath, 0755); err != nil {
		return fmt.Errorf("failed to create global templates directory: %w", err)
	}
	staticPath := filepath.Join(projectPath, "static")
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

	if !e.Cfg.SetupAllAuth {
		registrationPath := filepath.Join(globalTemplatesPath, "registration")
		if err := os.MkdirAll(registrationPath, 0755); err != nil {
			return fmt.Errorf("failed to create templates/registration directory: %w", err)
		}
		login := assets.DjangoLoginTemplate(e.Cfg.SetupTailwind)
		if err := os.WriteFile(filepath.Join(registrationPath, "login.html"), []byte(login), 0644); err != nil {
			return fmt.Errorf("failed to create registration/login.html: %w", err)
		}
	}

	for _, page := range assets.ErrorPages {
		name := page.Code + ".html"
		content := assets.ErrorTemplate(page, e.Cfg.SetupTailwind)
		if err := os.WriteFile(filepath.Join(globalTemplatesPath, name), []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to create %s: %w", name, err)
		}
	}

	if !e.Cfg.SetupTailwind && e.Cfg.UseGlobalTemplates {
		if err := os.MkdirAll(filepath.Join(staticPath, "css"), 0755); err != nil {
			return fmt.Errorf("failed to create static/css directory: %w", err)
		}

		if err := os.WriteFile(filepath.Join(staticPath, "css", "base.css"), []byte(assets.BaseCSS()), 0644); err != nil {
			return fmt.Errorf("failed to create base.css: %w", err)
		}

		if err := os.WriteFile(filepath.Join(staticPath, "css", "home.css"), []byte(assets.HomeCSS()), 0644); err != nil {
			return fmt.Errorf("failed to create home.css: %w", err)
		}

		if err := os.WriteFile(filepath.Join(staticPath, "css", "app.css"), []byte(assets.AppCSS()), 0644); err != nil {
			return fmt.Errorf("failed to create app.css: %w", err)
		}

		if err := os.WriteFile(filepath.Join(staticPath, "css", "auth.css"), []byte(assets.AuthCSS(e.Cfg.SetupAllAuth)), 0644); err != nil {
			return fmt.Errorf("failed to create auth.css: %w", err)
		}
	}

	e.Step("Created global templates and static files.")
	return nil
}

func (e *Env) mainJS() string {
	// base.html loads this as a classic script, so it must not use module
	// syntax: an "export" here is a SyntaxError on every page.
	if !e.Cfg.SetupHTMX {
		return "// Site-wide scripts. base.html loads this file on every page.\n"
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
	return fmt.Sprintf(`from typing import TypedDict

from django.http import HttpRequest

class ProjectContext(TypedDict):
    project_name: str

def project_context(request: HttpRequest) -> ProjectContext:
    return {
        'project_name': '%s',
    }
`, e.Cfg.ProjectName)
}

func (e *Env) writeProjectURLs(projectPath string) error {
	projectModulePath := filepath.Join(projectPath, e.Cfg.ModuleName())
	if e.Cfg.CreateTemplates {
		contextProcessorsPath := filepath.Join(projectModulePath, "context_processors.py")
		if err := os.WriteFile(contextProcessorsPath, []byte(e.renderContextProcessors()), 0644); err != nil {
			return fmt.Errorf("failed to create context_processors.py: %w", err)
		}
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
	if e.Cfg.SetupNinja {
		return "/api/v1/docs"
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

	nativeCSP := config.DjangoSupportsNativeCSP(e.Cfg.DjangoVersion)
	if e.Cfg.SetupRestFramework {
		lines = append(lines, "from drf_spectacular.views import SpectacularAPIView, SpectacularSwaggerSplitView")
		if nativeCSP {
			lines = append(lines, "from django.utils.csp import CSP", "from django.views.decorators.csp import csp_override, csp_report_only_override")
		} else {
			lines = append(lines, "from csp.constants import UNSAFE_INLINE", "from csp.decorators import csp_update")
		}
	}

	if e.Cfg.SetupNinja && e.Cfg.AppName != "" {
		lines = append(lines, fmt.Sprintf("from %s.api import api", e.Cfg.AppName))
	}

	lines = append(lines, "")
	lines = append(lines, "from . import views")

	if e.Cfg.SetupRestFramework {
		lines = append(lines, "", swaggerViewWithInlineStyles(nativeCSP))
	}

	lines = append(lines, "", "urlpatterns = [")

	if e.Cfg.CreateTemplates {
		lines = append(lines, "    path('', views.HomeView.as_view(), name='home'),")
	} else {
		lines = append(lines, fmt.Sprintf(
			"    path('', RedirectView.as_view(url='%s', permanent=False), name='home'),", e.apiOnlyHomeTarget()))
	}

	lines = append(lines, "    path(settings.ADMIN_URL, admin.site.urls),")
	lines = append(lines, "    path('healthz/', views.healthz, name='healthz'),")

	switch {
	case e.Cfg.SetupAllAuth:
		lines = append(lines, "    path('accounts/', include('allauth.urls')),")
	case e.Cfg.CreateTemplates:
		lines = append(lines, "    path('accounts/', include('django.contrib.auth.urls')),")
	}

	switch {
	case e.Cfg.SetupRestFramework:
		lines = append(lines, "    path('api/schema/', SpectacularAPIView.as_view(), name='schema'),")
		lines = append(lines, "    path('api/schema/swagger-ui/', swagger_ui, name='swagger-ui'),")
		lines = append(lines, "    path('api-auth/', include('rest_framework.urls', namespace='rest_framework')),")
		if e.Cfg.AppName != "" {
			lines = append(lines, fmt.Sprintf("    path('api/v1/', include('%s.api')),", e.Cfg.AppName))
			if !e.Cfg.APIOnly() {
				lines = append(lines, fmt.Sprintf("    path('%s/', include('%s.urls')),", e.Cfg.AppName, e.Cfg.AppName))
			}
		}

	case e.Cfg.SetupNinja && e.Cfg.AppName != "":
		lines = append(lines, "    path('api/v1/', api.urls),")
		if !e.Cfg.APIOnly() {
			lines = append(lines, fmt.Sprintf("    path('%s/', include('%s.urls')),", e.Cfg.AppName, e.Cfg.AppName))
		}

	case e.Cfg.AppName != "" && e.Cfg.CreateAppTemplates:
		lines = append(lines, fmt.Sprintf("    path('%s/', include('%s.urls', namespace='%s')),", e.Cfg.AppName, e.Cfg.AppName, e.Cfg.AppName))
	}

	lines = append(lines, "]")
	lines = append(lines, "")
	// Pages stacks keep Django's own error views: they render templates/404.html
	// and friends already, and render 500.html with an empty Context. API-only
	// projects have no templates, so they answer in JSON instead.
	if e.Cfg.APIOnly() {
		lines = append(lines, "handler400 = 'MODULE.views.bad_request'")
		lines = append(lines, "handler403 = 'MODULE.views.permission_denied'")
		lines = append(lines, "handler404 = 'MODULE.views.page_not_found'")
		lines = append(lines, "handler500 = 'MODULE.views.server_error'")
		lines = append(lines, "")
	}
	if e.Cfg.CreateTemplates {
		lines = append(lines, "if settings.DEBUG:")
		lines = append(lines, "    urlpatterns += [path('__reload__/', include('django_browser_reload.urls'))]")
	}

	return strings.ReplaceAll(strings.Join(lines, "\n"), "MODULE.views", e.Cfg.ModuleName()+".views")
}

// swaggerViewWithInlineStyles is drf-spectacular's Swagger page with inline
// styles allowed on that view alone: its template carries one <style> block,
// and every other page keeps style-src 'self'. Each policy is only extended
// when it is in force, so development still only reports.
func swaggerViewWithInlineStyles(nativeCSP bool) string {
	if nativeCSP {
		return `swagger_ui = SpectacularSwaggerSplitView.as_view(url_name='schema')
if settings.SECURE_CSP:
    swagger_ui = csp_override({**settings.SECURE_CSP, 'style-src': [CSP.SELF, CSP.UNSAFE_INLINE]})(swagger_ui)
if settings.SECURE_CSP_REPORT_ONLY:
    swagger_ui = csp_report_only_override(
        {**settings.SECURE_CSP_REPORT_ONLY, 'style-src': [CSP.SELF, CSP.UNSAFE_INLINE]},
    )(swagger_ui)`
	}
	return `swagger_ui = csp_update({'style-src': [UNSAFE_INLINE]}, REPORT_ONLY=True)(
    csp_update({'style-src': [UNSAFE_INLINE]})(SpectacularSwaggerSplitView.as_view(url_name='schema')),
)`
}
