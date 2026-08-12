package assets_test

import (
	"strings"
	"testing"

	"github.com/vaibdix/django-cli/internal/assets"
)

func everyPage() []assets.Page {
	return []assets.Page{assets.PageBase, assets.PageIndex, assets.PageAPIDocs}
}

func TestDjangoTemplateTagsSurvive(t *testing.T) {
	t.Parallel()

	for _, fs := range featureSets() {
		t.Run(fs.name, func(t *testing.T) {
			t.Parallel()

			base := assets.RenderPage(assets.PageBase, fs.opts)
			for _, tag := range []string{
				"{% block title %}", "{% block content %}", "{% block extra_head %}",
				"{% block extra_body %}", `{{ project_name|default:`, `{% now "Y" %}`,
			} {
				if !strings.Contains(base, tag) {
					t.Errorf("base.html lost %s", tag)
				}
			}

			index := assets.RenderPage(assets.PageIndex, fs.opts)
			for _, tag := range []string{"{% extends 'base.html' %}", "{% block title %}", "{% block content %}"} {
				if !strings.Contains(index, tag) {
					t.Errorf("index.html lost %s", tag)
				}
			}
		})
	}
}

func TestStaticLinksMatchTheRenderingMode(t *testing.T) {
	t.Parallel()

	linked := assets.RenderPage(assets.PageBase, assets.PageOptions{GlobalTemplates: true})
	if !strings.Contains(linked, "{% load static %}") {
		t.Error("the linked-stylesheet variant does not load the static tag library")
	}
	if !strings.Contains(linked, "{% static 'css/base.css' %}") {
		t.Error("the linked-stylesheet variant does not link base.css")
	}

	inlined := assets.RenderPage(assets.PageBase, assets.PageOptions{})
	if strings.Contains(inlined, "{% static 'css/base.css' %}") {
		t.Error("the inlined variant links a stylesheet that the project has no static dir for")
	}
	if !strings.Contains(inlined, "<style>") {
		t.Error("the inlined variant carries no stylesheet at all, so the page renders unstyled")
	}

	tailwind := assets.RenderPage(assets.PageBase, assets.PageOptions{Tailwind: true})
	for _, needle := range []string{
		"{% static 'dist/styles.css' %}", "{% static 'js/main.js' %}",
	} {
		if !strings.Contains(tailwind, needle) {
			t.Errorf("the tailwind base layout lost %s", needle)
		}
	}

	if strings.Contains(tailwind, "django_browser_reload") {
		t.Error("the base layout loads a tag library that production settings do not install, so every page would raise")
	}
}

func TestHTMXWiringIsAllOrNothing(t *testing.T) {
	t.Parallel()

	for _, tailwind := range []bool{false, true} {
		on := assets.RenderPage(assets.PageBase, assets.PageOptions{
			GlobalTemplates: true, HTMX: true, Tailwind: tailwind,
		})
		for _, needle := range []string{"{% load django_htmx %}", "{% htmx_script %}", "hx-headers", "X-CSRFToken"} {
			if !strings.Contains(on, needle) {
				t.Errorf("tailwind=%v: HTMX is enabled but base.html lost %s", tailwind, needle)
			}
		}

		off := assets.RenderPage(assets.PageBase, assets.PageOptions{
			GlobalTemplates: true, Tailwind: tailwind,
		})
		for _, needle := range []string{"django_htmx", "htmx_script", "hx-headers"} {
			if strings.Contains(off, needle) {
				t.Errorf("tailwind=%v: HTMX is disabled but base.html still carries %s", tailwind, needle)
			}
		}
	}
}

func TestAllAuthPagesKeepTheirFormContract(t *testing.T) {
	t.Parallel()

	pages := map[string]struct {
		render func(bool) string
		action string
		fields []string
	}{
		"login": {assets.AllAuthLogin, "account_login",
			[]string{`name="login"`, `id="id_login"`, `name="password"`, `id="id_password"`,
				"form.login.errors", "form.password.errors", "form.non_field_errors"}},
		"signup": {assets.AllAuthSignup, "account_signup",
			[]string{`name="username"`, `id="id_username"`, `name="email"`, `id="id_email"`,
				`name="password1"`, `id="id_password1"`, `name="password2"`, `id="id_password2"`,
				"form.username.errors", "form.email.errors"}},
		"logout": {assets.AllAuthLogout, "account_logout", nil},
	}

	for name, page := range pages {
		for _, tailwind := range []bool{false, true} {
			got := page.render(tailwind)

			if !strings.Contains(got, "{% csrf_token %}") {
				t.Errorf("%s (tailwind=%v): no csrf_token, so every POST is rejected", name, tailwind)
			}
			if !strings.Contains(got, "{% url '"+page.action+"' %}") {
				t.Errorf("%s (tailwind=%v): form does not post to {%% url '%s' %%}", name, tailwind, page.action)
			}
			for _, field := range page.fields {
				if !strings.Contains(got, field) {
					t.Errorf("%s (tailwind=%v): lost %s, which allauth binds to", name, tailwind, field)
				}
			}
			if name != "logout" && !strings.Contains(got, "redirect_field_name") {
				t.Errorf("%s (tailwind=%v): lost the redirect field, so ?next= is ignored after sign-in", name, tailwind)
			}
		}
	}
}

func TestNavigationFollowsTheFeatureFlags(t *testing.T) {
	t.Parallel()

	for _, tailwind := range []bool{false, true} {
		base := func(o assets.PageOptions) string {
			o.Tailwind = tailwind
			o.GlobalTemplates = true
			return assets.RenderPage(assets.PageBase, o)
		}

		if got := base(assets.PageOptions{API: true}); !strings.Contains(got, "{% url 'swagger-ui' %}") {
			t.Errorf("tailwind=%v: API is on but there is no swagger-ui link", tailwind)
		}
		if got := base(assets.PageOptions{}); strings.Contains(got, "swagger-ui") {
			t.Errorf("tailwind=%v: API is off but a swagger-ui link is still rendered", tailwind)
		}

		withAuth := base(assets.PageOptions{AllAuth: true})
		for _, needle := range []string{
			"{% if user.is_authenticated %}", "{% url 'account_logout' %}",
			"{% url 'account_login' %}", "{% url 'account_signup' %}",
		} {
			if !strings.Contains(withAuth, needle) {
				t.Errorf("tailwind=%v: AllAuth is on but the nav lost %s", tailwind, needle)
			}
		}
		if got := base(assets.PageOptions{}); strings.Contains(got, "account_login") {
			t.Errorf("tailwind=%v: AllAuth is off but the nav still links to it", tailwind)
		}

		withApp := base(assets.PageOptions{AppName: "catalog"})
		if !strings.Contains(withApp, "catalog:book_list") && !strings.Contains(withApp, "app_name|add:':book_list'") {
			t.Errorf("tailwind=%v: the nav does not link to the app by route name", tailwind)
		}
		if !tailwind && !strings.Contains(withApp, "{% if app_name %}") {
			t.Errorf("the plain nav renders the app link unconditionally; " +
				"a project with no app would show a link to nothing")
		}
	}
}

func TestEveryStandalonePageDeclaresTheFrame(t *testing.T) {
	t.Parallel()

	for _, fs := range featureSets() {
		for _, page := range everyPage() {
			got := assets.RenderPage(page, fs.opts)

			standalone := strings.Contains(got, "<!DOCTYPE html>")
			extends := strings.Contains(got, "{% extends 'base.html' %}")
			framed := strings.Contains(got, `class="frame"`)

			switch {
			case standalone && !framed:
				t.Errorf("%s/%s is a whole document but declares no framed column, so it renders full-bleed",
					fs.name, page)
			case extends && framed:
				t.Errorf("%s/%s extends base.html and declares its own frame; the rules would be drawn twice",
					fs.name, page)
			case !standalone && !extends:
				t.Errorf("%s/%s is neither a whole document nor extends base.html", fs.name, page)
			}
		}
	}
}

func TestNoPageBringsItsOwnStylesheet(t *testing.T) {
	t.Parallel()

	for _, set := range featureSets() {
		for _, page := range []assets.Page{assets.PageBase, assets.PageIndex} {
			rendered := assets.RenderPage(page, set.opts)

			block := extraHeadBlock(rendered)
			if block == "" {
				continue
			}
			if strings.Contains(block, "stylesheet") || strings.Contains(block, "<style") {
				t.Errorf("%v page (%s) adds styles in extra_head; boosted navigation drops them:\n%s",
					page, set.name, block)
			}
		}
	}
}

func TestBaseCarriesEveryPagesStylesheet(t *testing.T) {
	t.Parallel()

	base := assets.RenderPage(assets.PageBase, assets.PageOptions{GlobalTemplates: true, HTMX: true, AllAuth: true})
	for _, sheet := range []string{"css/base.css", "css/home.css", "css/app.css", "css/auth.css"} {
		if !strings.Contains(base, sheet) {
			t.Errorf("base.html does not link %s, so pages that need it break on boosted navigation", sheet)
		}
	}

	inlined := assets.RenderPage(assets.PageBase, assets.PageOptions{HTMX: true, AllAuth: true})
	for _, marker := range []string{".auth-card", ".book-card", ".hero"} {
		if !strings.Contains(inlined, marker) {
			t.Errorf("the inlined base.html is missing the %s rules", marker)
		}
	}
}

func extraHeadBlock(rendered string) string {
	start := strings.Index(rendered, "{% block extra_head %}")
	if start == -1 {
		return ""
	}
	end := strings.Index(rendered[start:], "{% endblock %}")
	if end == -1 {
		return ""
	}
	return rendered[start : start+end]
}
