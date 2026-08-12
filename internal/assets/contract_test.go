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

func TestHTMXCSRFHeaderRefreshesAfterLogin(t *testing.T) {
	t.Parallel()

	for _, tailwind := range []bool{false, true} {
		base := assets.RenderPage(assets.PageBase, assets.PageOptions{
			GlobalTemplates: true, HTMX: true, Tailwind: tailwind,
		})

		bodyStart := strings.Index(base, "<body")
		if bodyStart == -1 {
			t.Fatalf("tailwind=%v: base.html has no body", tailwind)
		}
		bodyEnd := bodyStart + strings.Index(base[bodyStart:], ">")
		if strings.Contains(base[bodyStart:bodyEnd], "hx-headers") {
			t.Errorf("tailwind=%v: the persistent body keeps the CSRF token after login", tailwind)
		}
		if !strings.Contains(base, `<div class="frame" hx-headers='{"X-CSRFToken": "{{ csrf_token }}"}'>`) {
			t.Errorf("tailwind=%v: the replaceable frame does not carry the current CSRF token", tailwind)
		}
	}
}

func TestAllAuthPagesRenderWhateverFieldsAllAuthGivesThem(t *testing.T) {
	t.Parallel()

	for _, page := range assets.AuthPages {
		for _, tailwind := range []bool{false, true} {
			got := assets.AuthFormTemplate(page, tailwind)

			if !strings.Contains(got, "{% csrf_token %}") {
				t.Errorf("%s (tailwind=%v): no csrf_token, so every POST is rejected", page.Name, tailwind)
			}
			for _, want := range []string{
				"{% for field in form %}",
				"{{ field.label_tag }}",
				"{{ field }}",
				"{% for error in field.errors %}",
				"{% for error in form.non_field_errors %}",
				"redirect_field_name",
			} {
				if !strings.Contains(got, want) {
					t.Errorf("%s (tailwind=%v): missing %s", page.Name, tailwind, want)
				}
			}
			if strings.Contains(got, `name="password1"`) || strings.Contains(got, `name="login"`) {
				t.Errorf("%s (tailwind=%v): still hardcodes allauth field names, which drift with its settings",
					page.Name, tailwind)
			}
			if page.Action != "" && !strings.Contains(got, "{% url '"+page.Action+"' %}") {
				t.Errorf("%s (tailwind=%v): form does not post to {%% url '%s' %%}", page.Name, tailwind, page.Action)
			}
		}
	}
}

func TestAllAuthShipsThePagesItsFlowsLinkTo(t *testing.T) {
	t.Parallel()

	names := map[string]bool{}
	for _, page := range assets.AuthPages {
		names[page.Name] = true
	}
	for _, page := range assets.AuthNotices {
		names[page.Name] = true
	}

	for _, required := range []string{
		"login", "signup", "password_reset", "password_reset_done",
		"password_reset_from_key", "password_reset_from_key_done",
		"verification_sent", "email_confirm", "password_change",
	} {
		if !names[required] {
			t.Errorf("allauth links to %s.html but the scaffold does not ship it, so that step 500s", required)
		}
	}
}

func TestLogoutStillPostsToAllAuth(t *testing.T) {
	t.Parallel()

	for _, tailwind := range []bool{false, true} {
		got := assets.AllAuthLogout(tailwind)
		if !strings.Contains(got, "{% csrf_token %}") || !strings.Contains(got, "{% url 'account_logout' %}") {
			t.Errorf("logout (tailwind=%v) cannot submit:\n%s", tailwind, got)
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

func TestAuthPagesDoNotDumpTheirValidatorHelpText(t *testing.T) {
	t.Parallel()

	for _, tailwind := range []bool{false, true} {
		for _, page := range assets.AuthPages {
			body := assets.AuthFormTemplate(page, tailwind)

			if strings.Contains(body, "field.help_text") {
				t.Errorf("tailwind=%v %s renders field.help_text; on signup that is the whole "+
					"AUTH_PASSWORD_VALIDATORS list, which buries the form:\n%s",
					tailwind, page.Name, body)
			}

			if !strings.Contains(body, "field.errors") {
				t.Errorf("tailwind=%v %s renders no field errors, so a rejected password "+
					"is never explained:\n%s", tailwind, page.Name, body)
			}
		}
	}
}
