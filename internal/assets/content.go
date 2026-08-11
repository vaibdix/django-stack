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
	return `<nav><a href="/">Home</a>` +
		`{% if app_name %}<a href="/{{ app_name }}/">{{ app_name|title }}</a>{% endif %}` +
		apiDocsLink(opts.API) +
		`<a href="/admin/">Admin</a>` +
		authNavLinks(opts.AllAuth, opts.Tailwind) + `</nav>`
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
                    <svg width="24" height="24" viewBox="0 0 32 32" fill="none" aria-hidden="true">
                        <rect width="32" height="32" rx="6" fill="currentColor" opacity="0.12"/>
                        <path d="M12 8L20 16L12 24" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"/>
                    </svg>
                    <h1>{{ project_name|default:"Django" }}</h1>
                </a>
                `
}

// pageFooter closes the framed column: the page body, the two-up closing band,
// and the copyright line.
//
// The reference has an 88px #spacer here, which gives its column a bottom edge
// because nothing follows it. This layout ends in a footer, which does the same
// job, so carrying both left the page scrolling past an empty block.
func pageFooter() string {
	return `    <main>{% block content %}{% endblock %}</main>
    <section id="next-steps" class="ticks">
        <div id="docs">
            <svg class="icon" viewBox="0 0 22 22" fill="none" stroke="currentColor" stroke-width="1.6" aria-hidden="true"><path d="M4 3h9l5 5v11H4z"/><path d="M13 3v5h5"/></svg>
            <h2>Documentation</h2>
            <p>Your questions, answered</p>
            <ul>
                <li><a href="https://docs.djangoproject.com/">Explore Django</a></li>
                <li><a href="/admin/">Open admin</a></li>
            </ul>
        </div>
        <div id="build">
            <svg class="icon" viewBox="0 0 22 22" fill="none" stroke="currentColor" stroke-width="1.6" aria-hidden="true"><path d="M11 2l7 4v10l-7 4-7-4V6z"/><path d="M11 11l7-4M11 11v9M11 11L4 7"/></svg>
            <h2>Start building</h2>
            <p>Register a model and it appears in the admin</p>
            <ul>
                <li><a href="https://docs.djangoproject.com/en/stable/intro/tutorial01/">Tutorial</a></li>
                <li><a href="https://docs.djangoproject.com/en/stable/ref/models/">Model reference</a></li>
            </ul>
        </div>
    </section>
    <div class="ticks"></div>
    <footer>
        <div class="footer-bottom"><p>© {% now "Y" %} {{ project_name|default:"Django Site" }}</p></div>
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
<section id="center">
    <div>
        <h1>Get started</h1>
        <p class="hero-subtitle">Edit <code>{{ project_name|default:"project" }}/views.py</code> and save to see your changes</p>
    </div>
    <span class="badge">Django is running</span>
    <div class="button-group">
        <a href="https://docs.djangoproject.com/" class="button button-primary">Read the docs</a>
        <a href="/admin/" class="button button-secondary">Open admin</a>
    </div>
</section>
<div class="ticks"></div>
<section class="features">
    <div>
        <h3 class="feature-title">Fast development</h3>
        <p class="feature-description">A batteries-included framework: the admin, the ORM and the auth system are already here.</p>
    </div>
    <div>
        <h3 class="feature-title">Security first</h3>
        <p class="feature-description">Argon2 hashing, CSRF protection and secure cookie defaults are configured from the first run.</p>
    </div>
</section>
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
    <main class="container">
        <h1>API Documentation</h1>
        <p>Explore the browsable API, the generated schema and the admin from one place.</p>
    </main>
  </div>
</body>
</html>`
}
