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
	return render("main.js.tmpl", struct{ HTMX bool }{p.cfg.UsesHTMX()})
}

func (p project) renderContextProcessors() string {
	return render("context_processors.py.tmpl", struct {
		LazyAnnotations bool
		Name            string
	}{p.lazyAnnotations(), p.cfg.Name().String()})
}

func (e *Env) writeProjectURLs(ctx context.Context) error {
	files, err := e.urlFiles()
	if err != nil {
		return err
	}
	return e.emit(files)
}

func (p project) urlFiles() (fileSet, error) {
	urls, err := p.buildProjectURLsContent()
	if err != nil {
		return nil, err
	}
	files := fileSet{}
	module := p.cfg.ModuleName()
	files.add(filepath.Join(module, "views.py"), p.buildProjectViewsContent(), fileMode)
	files.add(filepath.Join(module, urlsFile), urls, fileMode)
	if p.cfg.CreateTemplates() {
		files.add(filepath.Join(module, "context_processors.py"), p.renderContextProcessors(), fileMode)
	}
	return files, nil
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

func (p project) urlRoutes(urls []urlContribution) []route {
	routes := []route{
		{rank: rankAdminRoute, line: "    path(settings.ADMIN_URL, admin.site.urls),"},
		{rank: rankHealthRoute, line: `    path("healthz/", views.healthz, name="healthz"),`},
	}
	if p.cfg.CreateTemplates() {
		routes = append(routes, route{rank: rankHomeRoute, line: `    path("", views.HomeView.as_view(), name="home"),`})
	} else {
		routes = append(routes, route{rank: rankHomeRoute, line: fmt.Sprintf(
			`    path("", RedirectView.as_view(url="%s", permanent=False), name="home"),`, apiOnlyHomeTarget(urls))})
	}
	if p.cfg.CreateTemplates() && !anyURL(urls, func(u urlContribution) bool { return u.auth }) {
		routes = append(routes, route{rank: rankAuthRoute, line: `    path("accounts/", include("django.contrib.auth.urls")),`})
	}
	if p.cfg.App().Ok() && p.cfg.CreateTemplates() && !anyURL(urls, func(u urlContribution) bool { return u.appRoute }) {
		app := p.cfg.App().String()
		routes = append(routes, route{rank: rankAppRoute, line: fmt.Sprintf(`    path("%s/", include("%s.urls", namespace="%s")),`, app, app, app)})
	}
	for _, u := range urls {
		routes = append(routes, u.routes...)
	}
	return routes
}

func (p project) buildProjectURLsContent() (string, error) {
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
	if p.cfg.APIOnly() {
		views := p.cfg.ModuleName() + ".views"
		body = append(body,
			`handler400 = "`+views+`.bad_request"`,
			`handler403 = "`+views+`.permission_denied"`,
			`handler404 = "`+views+`.page_not_found"`,
			`handler500 = "`+views+`.server_error"`,
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
