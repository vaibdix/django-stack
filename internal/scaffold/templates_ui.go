package scaffold

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/vaibdix/django-stack/internal/assets"
	"github.com/vaibdix/django-stack/internal/config"
)

func (e *Env) applyGlobalTemplates() error {
	if err := e.emit(e.templateFiles()); err != nil {
		return err
	}
	e.Step("Created global templates and static files.")
	return nil
}

func (p project) templateFiles() fileSet {
	files := fileSet{}
	if !p.cfg.CreateTemplates() {
		return files
	}
	files.add(filepath.Join("static", "js", "main.js"), p.mainJS(), fileMode)

	opts := p.pageOptions()
	templates := map[string]string{
		"base.html":  assets.RenderPage(assets.PageBase, opts),
		"index.html": assets.RenderPage(assets.PageIndex, opts),
	}
	if !p.cfg.Has(config.AllAuth) {
		templates[filepath.Join("registration", "login.html")] = assets.DjangoLoginTemplate(p.cfg.UsesTailwind())
	}
	for _, page := range assets.ErrorPages {
		templates[page.Code+".html"] = assets.ErrorTemplate(page, p.cfg.UsesTailwind())
	}
	files.addAll(templatesDir, templates)

	if !p.cfg.UsesTailwind() {
		styles := map[string]string{
			"base.css": assets.BaseCSS(),
			"home.css": assets.HomeCSS(),
			"app.css":  assets.AppCSS(),
			"auth.css": assets.AuthCSS(p.cfg.Has(config.AllAuth)),
		}
		files.addAll(filepath.Join("static", "css"), styles)
	}
	return files
}

func (p project) mainJS() string {
	if !p.cfg.UsesHTMX() {
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

func (p project) renderContextProcessors() string {
	imports := "from typing import TypedDict\n\nfrom django.http import HttpRequest\n"
	if p.lazyAnnotations() {
		imports = "from typing import TYPE_CHECKING\nfrom typing import TypedDict\n\nif TYPE_CHECKING:\n    from django.http import HttpRequest\n"
	}
	return imports + fmt.Sprintf(`

class ProjectContext(TypedDict):
    project_name: str


def project_context(request: HttpRequest) -> ProjectContext:
    return {
        "project_name": "%s",
    }
`, p.cfg.Name)
}

func (e *Env) writeProjectURLs(ctx context.Context) error {
	return e.emit(e.urlFiles())
}

func (p project) urlFiles() fileSet {
	files := fileSet{}
	module := p.cfg.ModuleName()
	files.add(filepath.Join(module, "views.py"), p.buildProjectViewsContent(), fileMode)
	files.add(filepath.Join(module, urlsFile), p.buildProjectURLsContent(), fileMode)
	if p.cfg.CreateTemplates() {
		files.add(filepath.Join(module, "context_processors.py"), p.renderContextProcessors(), fileMode)
	}
	return files
}

func (p project) buildProjectViewsContent() string {
	return render("project_views.py.tmpl", struct{ Templates bool }{Templates: p.cfg.CreateTemplates()})
}

func apiOnlyHomeTarget(urls []urlContribution) string {
	for _, u := range urls {
		if u.home != "" {
			return u.home
		}
	}
	return "/admin/"
}

func (p project) buildProjectURLsContent() string {
	urls := p.urlContributions()
	var body []string
	for _, u := range urls {
		for _, block := range u.blocks {
			body = append(body, "", block)
		}
	}

	body = append(body, "", "urlpatterns = [")
	if p.cfg.CreateTemplates() {
		body = append(body, `    path("", views.HomeView.as_view(), name="home"),`)
	} else {
		body = append(body, fmt.Sprintf(
			`    path("", RedirectView.as_view(url="%s", permanent=False), name="home"),`, apiOnlyHomeTarget(urls)))
	}
	body = append(body, "    path(settings.ADMIN_URL, admin.site.urls),")
	body = append(body, `    path("healthz/", views.healthz, name="healthz"),`)
	if p.cfg.CreateTemplates() && !anyURL(urls, func(u urlContribution) bool { return u.auth }) {
		body = append(body, `    path("accounts/", include("django.contrib.auth.urls")),`)
	}
	for _, u := range urls {
		body = append(body, u.patterns...)
	}
	if p.cfg.App != "" && p.cfg.CreateTemplates() && !anyURL(urls, func(u urlContribution) bool { return u.appRoute }) {
		body = append(body, fmt.Sprintf(`    path("%s/", include("%s.urls", namespace="%s")),`, p.cfg.App, p.cfg.App, p.cfg.App))
	}
	body = append(body, "]", "")
	if p.cfg.APIOnly() {
		body = append(body,
			`handler400 = "MODULE.views.bad_request"`,
			`handler403 = "MODULE.views.permission_denied"`,
			`handler404 = "MODULE.views.page_not_found"`,
			`handler500 = "MODULE.views.server_error"`,
			"")
	}
	if p.cfg.CreateTemplates() {
		body = append(body, "if settings.DEBUG:",
			`    urlpatterns += [path("__reload__/", include("django_browser_reload.urls"))]`, "")
	}

	imports := []string{
		"from django.conf import settings",
		"from django.contrib import admin",
		"from django.urls import path",
		"from . import views",
	}
	if strings.Contains(strings.Join(body, "\n"), "include(") {
		imports = append(imports, "from django.urls import include")
	}
	if p.cfg.APIOnly() {
		imports = append(imports, "from django.views.generic import RedirectView")
	}
	for _, u := range urls {
		for _, line := range u.imports {
			imports = append(imports, singleImports(line)...)
		}
	}

	lines := append(orderImports(imports, p.firstPartyModules()), body...)
	return strings.ReplaceAll(strings.Join(lines, "\n"), "MODULE.views", p.cfg.ModuleName()+".views")
}

func singleImports(line string) []string {
	module, members, ok := strings.Cut(strings.TrimPrefix(line, "from "), " import ")
	if !ok || !strings.HasPrefix(line, "from ") {
		return []string{line}
	}
	var out []string
	for member := range strings.SplitSeq(members, ",") {
		out = append(out, "from "+module+" import "+strings.TrimSpace(member))
	}
	return out
}
