package assets

import "strings"

// RenderPage renders one of the built-in HTML pages for the given feature set.
//
// An unrecognised page returns the empty string; callers write the result to
// disk, and empty output is easier to spot than plausible-looking markup for
// the wrong page.
func RenderPage(page Page, opts PageOptions) string {
	switch page {
	case PageBase:
		return renderBasePage(opts)
	case PageIndex:
		return renderIndexPage(opts)
	case PageAPIDocs:
		return renderAPIDocsPage(opts)
	default:
		return ""
	}
}

// apiDocsLink is the inline nav entry used by the non-Tailwind layouts.
func apiDocsLink(enabled bool) string {
	if !enabled {
		return ""
	}
	return `<a href="/api/schema/swagger-ui/">API Docs</a>`
}

// htmxTags holds the three fragments HTMX needs in the base layout. All three
// are emitted together or not at all — a page with the script but no CSRF
// header fails at runtime on the first POST.
type htmxTags struct {
	Load     string
	Script   string
	BodyAttr string
}

func htmxTagsFor(enabled bool) htmxTags {
	if !enabled {
		return htmxTags{}
	}
	return htmxTags{
		Load:     "\n    {% load django_htmx %}",
		Script:   "\n    {% htmx_script %}",
		BodyAttr: ` hx-headers='{"X-CSRFToken": "{{ csrf_token }}"}'`,
	}
}

// sharedNav is the nav bar used by the plain and global-template layouts.
func sharedNav(opts PageOptions) string {
	return `<nav><a href="/">Home</a><a href="/admin/">Admin</a>` +
		`{% if app_name %}<a href="/{{ app_name }}/">{{ app_name|title }}</a>{% endif %}` +
		apiDocsLink(opts.API) + authNavLinks(opts.AllAuth, opts.Tailwind) + `</nav>`
}

// pageHeader is the masthead shared by the base layout's two variants. The two
// differed only in how the stylesheet was linked, so everything below the
// <head> is generated once.
//
// The accent-filled mark matches the CLI's own wordmark glyph; the arrow is
// knocked out in the page background colour rather than in black, so the two
// stay in step if the palette moves.
func pageHeader() string {
	return `  <div class="frame">
    <header>
        <div class="container">
            <div class="header-content">
                <a href="/" class="logo">
                    <svg width="28" height="28" viewBox="0 0 32 32" fill="none">
                        <rect width="32" height="32" rx="8" fill="#c084fc"/>
                        <path d="M12 8L20 16L12 24" stroke="#16171d" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
                    </svg>
                    <h1>{{ project_name|default:"Django" }}</h1>
                </a>
                `
}

// pageFooter closes the framed column: the page body, the two-up closing band,
// a defined bottom edge, and the copyright line.
func pageFooter() string {
	return `    <main>{% block content %}{% endblock %}</main>
    <section id="next-steps" class="ticks">
        <section>
            <span class="next-steps-icon">◆</span>
            <h2>Start with the admin</h2>
            <p>Every model you register shows up there immediately, with no extra wiring.</p>
            <div class="pill-row"><a class="pill" href="/admin/">Open admin</a></div>
        </section>
        <section>
            <span class="next-steps-icon">◈</span>
            <h2>Read the documentation</h2>
            <p>Django's own docs are the fastest route from this scaffold to a real feature.</p>
            <div class="pill-row"><a class="pill" href="https://docs.djangoproject.com/">Django docs</a></div>
        </section>
    </section>
    <div id="spacer"></div>
    <footer>
        <div class="footer-bottom"><p>© {% now "Y" %} {{ project_name|default:"Django Site" }}. All rights reserved.</p></div>
    </footer>
  </div>
    {% block extra_body %}{% endblock %}
</body>
</html>`
}

func renderBasePage(opts PageOptions) string {
	if opts.Tailwind {
		return tailwindBaseTemplateContent(opts.AppName, opts.API, opts.HTMX, opts.AllAuth)
	}

	htmx := htmxTagsFor(opts.HTMX)

	// The only difference between the two variants is the stylesheet: with a
	// static/ directory the CSS is linked, otherwise it is inlined.
	var styles string
	if opts.GlobalTemplates {
		styles = `
    {% load static %}` + htmx.Load + `
    <link rel="stylesheet" href="{% static 'css/base.css' %}">` + htmx.Script
	} else {
		styles = htmx.Load + `
    <style>` + baseCSS + `</style>` + htmx.Script
	}

	var b strings.Builder
	b.WriteString(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{% block title %}{{ project_name|default:"Django Site" }}{% endblock %}</title>`)
	b.WriteString(styles)
	b.WriteString(`
    {% block extra_head %}{% endblock %}
</head>
<body` + htmx.BodyAttr + `>
`)
	b.WriteString(pageHeader())
	b.WriteString(sharedNav(opts))
	b.WriteString(`
            </div>
        </div>
    </header>
`)
	b.WriteString(pageFooter())
	return b.String()
}

// heroAndFeatures is the landing-page body, identical across both non-Tailwind
// variants; only the stylesheet block above it differs.
func heroAndFeatures() string {
	return `{% block content %}
<div class="hero">
    <div class="hero-gradient"></div>
    <div class="hero-content">
        <div class="badge"><span>Production ready Django application</span></div>
        <h1 class="hero-title"><span>Ship Django products with a</span><span class="gradient-text">cleaner first impression.</span></h1>
        <p class="hero-subtitle">Django provides everything you need — <strong>the admin, the ORM and the auth system are already here.</strong></p>
        <div class="button-group">
            <a href="https://docs.djangoproject.com/" class="button button-primary">Get Started</a>
            <a href="/admin/" class="button button-secondary">Admin Panel</a>
        </div>
    </div>
</div>
<div class="features">
    <div class="features-header"><h2 class="features-title">Why Django?</h2></div>
    <div class="features-grid">
        <div class="feature-card"><h3 class="feature-title">Fast development</h3><p class="feature-description">A batteries-included framework: the admin, the ORM and the auth system are already here.</p></div>
        <div class="feature-card"><h3 class="feature-title">Security first</h3><p class="feature-description">Argon2 hashing, CSRF protection and secure cookie defaults are configured from the first run.</p></div>
    </div>
</div>
{% endblock %}`
}

func renderIndexPage(opts PageOptions) string {
	if opts.Tailwind {
		return tailwindIndexTemplateContent()
	}

	head := `{% extends 'base.html' %}
`
	if opts.GlobalTemplates {
		head += `{% load static %}
{% block title %}{{ project_name }} - Home{% endblock %}
{% block extra_head %}
<link rel="stylesheet" href="{% static 'css/home.css' %}">
{% endblock %}
`
	} else {
		head += `{% block title %}{{ project_name }} - Home{% endblock %}
{% block extra_head %}
<style>` + homeCSS + `</style>
{% endblock %}
`
	}

	return head + heroAndFeatures()
}

func renderAPIDocsPage(opts PageOptions) string {
	if opts.Tailwind {
		return tailwindApiDocsTemplateContent()
	}

	return `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{ project_name }} - API Documentation</title>
    <style>` + baseCSS + `</style>
</head>
<body>
  <div class="frame">
    <header>
        <div class="container">
            <div class="header-content">
                <a href="/" class="logo"><h1>{{ project_name }}</h1></a>
                <nav><a href="/">Home</a><a href="/admin/">Admin</a>{% if app_name %}<a href="/{{ app_name }}/">{{ app_name|title }}</a>{% endif %}` + apiDocsLink(opts.API) + `</nav>
            </div>
        </div>
    </header>
    <main class="container" style="padding: 3.5rem 0;">
        <h1 style="color: var(--heading); font-size: 1.75rem; letter-spacing: -0.03em; margin-bottom: 0.75rem;">API Documentation</h1>
        <p>Explore the browsable API, the generated schema and the admin from one place.</p>
    </main>
    <div id="spacer"></div>
  </div>
</body>
</html>`
}
