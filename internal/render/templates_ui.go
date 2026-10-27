package render

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/vaibdix/django-stack/internal/assets"
	"github.com/vaibdix/django-stack/internal/config"
)

func (p Project) templateFiles() FileSet {
	files := FileSet{}
	if !p.Cfg.CreateTemplates() {
		return files
	}
	files.add(filepath.Join("static", "js", "main.js"), p.mainJS(), FileMode)

	opts := p.pageOptions()
	templates := map[string]string{
		"base.html":  assets.RenderPage(assets.PageBase, opts),
		"index.html": assets.RenderPage(assets.PageIndex, opts),
	}
	if !p.Cfg.Has(config.AllAuth) {
		templates[filepath.Join("registration", "login.html")] = assets.DjangoLoginTemplate(p.Cfg.UsesTailwind())
	}
	for _, page := range assets.ErrorPages {
		templates[page.Code+".html"] = assets.ErrorTemplate(page, p.Cfg.UsesTailwind())
	}
	files.addAll(templatesDir, templates)

	if !p.Cfg.UsesTailwind() {
		styles := map[string]string{
			"base.css": assets.BaseCSS(),
			"home.css": assets.HomeCSS(),
			"app.css":  assets.AppCSS(),
			"auth.css": assets.AuthCSS(p.Cfg.Has(config.AllAuth)),
		}
		files.addAll(filepath.Join("static", "css"), styles)
	}
	return files
}

func (p Project) mainJS() string {
	return Render("main.js.tmpl", struct{ HTMX bool }{p.Cfg.UsesHTMX()})
}

func (p Project) renderContextProcessors() string {
	return Render("context_processors.py.tmpl", struct {
		LazyAnnotations bool
		Name            string
	}{p.lazyAnnotations(), p.Cfg.Name().String()})
}

func (p Project) urlFiles() (FileSet, error) {
	urls, err := p.buildProjectURLsContent()
	if err != nil {
		return nil, err
	}
	files := FileSet{}
	module := p.Cfg.ModuleName()
	files.add(filepath.Join(module, "views.py"), p.buildProjectViewsContent(), FileMode)
	files.add(filepath.Join(module, urlsFile), urls, FileMode)
	if p.Cfg.CreateTemplates() {
		files.add(filepath.Join(module, "context_processors.py"), p.renderContextProcessors(), FileMode)
	}
	return files, nil
}

func (p Project) buildProjectViewsContent() string {
	return Render("project_views.py.tmpl", struct{ Templates bool }{Templates: p.Cfg.CreateTemplates()})
}

func apiOnlyHomeTarget(urls []urlContribution) string {
	for _, u := range urls {
		if u.home != "" {
			return u.home
		}
	}
	return "/admin/"
}

func (p Project) urlRoutes(urls []urlContribution) []route {
	routes := []route{
		{rank: rankAdminRoute, line: "    path(settings.ADMIN_URL, admin.site.urls),"},
		{rank: rankHealthRoute, line: `    path("healthz/", views.healthz, name="healthz"),`},
	}
	if p.Cfg.CreateTemplates() {
		routes = append(routes, route{rank: rankHomeRoute, line: `    path("", views.HomeView.as_view(), name="home"),`})
	} else {
		routes = append(routes, route{rank: rankHomeRoute, line: fmt.Sprintf(
			`    path("", RedirectView.as_view(url="%s", permanent=False), name="home"),`, apiOnlyHomeTarget(urls))})
	}
	if p.Cfg.CreateTemplates() && !anyURL(urls, func(u urlContribution) bool { return u.auth }) {
		routes = append(routes, route{rank: rankAuthRoute, line: `    path("accounts/", include("django.contrib.auth.urls")),`})
	}
	if p.Cfg.App().Ok() && p.Cfg.CreateTemplates() && !anyURL(urls, func(u urlContribution) bool { return u.appRoute }) {
		app := p.Cfg.App().String()
		routes = append(routes, route{rank: rankAppRoute, line: fmt.Sprintf(`    path("%s/", include("%s.urls", namespace="%s")),`, app, app, app)})
	}
	for _, u := range urls {
		routes = append(routes, u.routes...)
	}
	return routes
}

func (p Project) buildProjectURLsContent() (string, error) {
	urls := p.urlContributions()
	var blocks []block
	for _, u := range urls {
		blocks = append(blocks, u.blocks...)
	}
	blocks, err := orderBlocks("URL", blocks)
	if err != nil {
		return "", err
	}
	routes, err := orderRoutes(p.urlRoutes(urls))
	if err != nil {
		return "", err
	}
	var body []string
	for _, b := range blocks {
		body = append(body, "", b.text)
	}
	body = append(body, "", "urlpatterns = [")
	body = append(body, routes...)
	body = append(body, "]", "")
	if p.Cfg.APIOnly() {
		views := p.Cfg.ModuleName() + ".views"
		body = append(body,
			`handler400 = "`+views+`.bad_request"`,
			`handler403 = "`+views+`.permission_denied"`,
			`handler404 = "`+views+`.page_not_found"`,
			`handler500 = "`+views+`.server_error"`,
			"")
	}
	if p.Cfg.CreateTemplates() {
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
	if p.Cfg.APIOnly() {
		imports = append(imports, "from django.views.generic import RedirectView")
	}
	for _, u := range urls {
		for _, line := range u.imports {
			imports = append(imports, singleImports(line)...)
		}
	}

	lines := append(orderImports(imports, p.firstPartyModules()), body...)
	return strings.Join(lines, "\n"), nil
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
