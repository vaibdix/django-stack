package assets_test

import (
	"strings"
	"testing"

	"github.com/vaibdix/django-cli/internal/assets"
)

// What the generated pages must contain, whatever they look like.
//
// The goldens pin bytes, which sounds like enough and is not: regenerate them
// after an edit that dropped an hx-target and the golden records the broken
// page as the new truth. Nothing fails, and the defect ships. That is the
// lesson lint_generated_test.go already learned for the generated Python.
//
// Every string below is a contract with something outside this repository —
// Django's template language, django-htmx's attribute names, allauth's form
// field names — so none of them can be renamed to suit a design.

func everyPage() []assets.Page {
	return []assets.Page{assets.PageBase, assets.PageIndex, assets.PageAPIDocs}
}

// TestDjangoTemplateTagsSurvive checks the tags that make the page a Django
// template rather than a static file. A dropped {% block content %} produces a
// layout that renders every child page as an empty shell.
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

// The stylesheet and script links. With GlobalTemplates the CSS is served from
// static/; without it the same CSS is inlined. Both paths are reachable and
// only one of them is exercised by a default scaffold.
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
		"{% load django_browser_reload %}", "{{ django_browser_reload_script }}",
	} {
		if !strings.Contains(tailwind, needle) {
			t.Errorf("the tailwind base layout lost %s", needle)
		}
	}
}

// HTMX needs three things together in the layout: the tag library, the script,
// and the CSRF header. A page with the script and no header fails on the first
// POST, at runtime, in the browser.
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

// allauth binds to these names. A restyle that renames a field or drops an
// error block produces a form that posts nothing allauth recognises, and the
// user sees a login page that silently refuses every password.
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

// The nav links each feature adds, in both directions. These are driven by
// PageOptions flags rather than by anything observable in the markup, so a
// wrong flag reads as a missing link and nothing else.
func TestNavigationFollowsTheFeatureFlags(t *testing.T) {
	t.Parallel()

	for _, tailwind := range []bool{false, true} {
		base := func(o assets.PageOptions) string {
			o.Tailwind = tailwind
			o.GlobalTemplates = true
			return assets.RenderPage(assets.PageBase, o)
		}

		if got := base(assets.PageOptions{API: true}); !strings.Contains(got, "/api/schema/swagger-ui/") {
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

		// The two variants link the app differently and both are correct: the
		// Tailwind nav interpolates the name when the project is generated,
		// the plain nav defers to the app_name context variable so it follows
		// the context processor at render time. What must hold either way is
		// that a nav entry for the app exists at all.
		withApp := base(assets.PageOptions{AppName: "catalog"})
		if !strings.Contains(withApp, "catalog") && !strings.Contains(withApp, "{{ app_name }}") {
			t.Errorf("tailwind=%v: the nav links to the app neither by name nor through app_name", tailwind)
		}
		if !tailwind && !strings.Contains(withApp, "{% if app_name %}") {
			t.Errorf("the plain nav renders the app link unconditionally; " +
				"a project with no app would show a link to nothing")
		}
	}
}

// The framed column is what gives the site its side rules. Missing it is not a
// crash — the page renders full-bleed and looks like a different product.
//
// Only a page that is a whole document has to declare the frame. A page that
// extends base.html inherits it, and declaring it again would nest one frame
// inside another and draw the rules twice.
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
