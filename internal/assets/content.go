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
func pageHeader() string {
	return `    <header>
        <div class="container">
            <div class="header-content">
                <a href="/" class="logo">
                    <svg width="32" height="32" viewBox="0 0 32 32" fill="none">
                        <rect width="32" height="32" rx="8" fill="white"/>
                        <path d="M12 8L20 16L12 24" stroke="black" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
                    </svg>
                    <h1>{{ project_name|default:"Django" }}</h1>
                </a>
                `
}

func pageFooter() string {
	return `    <main>{% block content %}{% endblock %}</main>
    <footer>
        <div class="container">
            <div class="footer-bottom"><p>© {% now "Y" %} {{ project_name|default:"Django Site" }}. All rights reserved.</p></div>
        </div>
    </footer>
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
        <h1 class="hero-title"><span>The Django</span><span class="gradient-text">Framework</span></h1>
        <p class="hero-subtitle">Django provides everything you need.</p>
        <div class="button-group">
            <a href="https://docs.djangoproject.com/" class="button button-primary">Get Started</a>
            <a href="/admin/" class="button button-secondary">Admin Panel</a>
        </div>
    </div>
</div>
<div class="features">
    <div class="features-header"><h2 class="features-title">Why Django?</h2></div>
    <div class="features-grid">
        <div class="feature-card"><h3 class="feature-title">Fast Development</h3><p class="feature-description">Batteries-included approach.</p></div>
        <div class="feature-card"><h3 class="feature-title">Security First</h3><p class="feature-description">Built-in protection.</p></div>
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
    <header>
        <div class="container">
            <div class="header-content">
                <a href="/" class="logo"><h1>{{ project_name }}</h1></a>
                <nav><a href="/">Home</a><a href="/admin/">Admin</a>{% if app_name %}<a href="/{{ app_name }}/">{{ app_name|title }}</a>{% endif %}` + apiDocsLink(opts.API) + `</nav>
            </div>
        </div>
    </header>
    <main class="container" style="padding: 4rem 1rem;">
        <h1>API Documentation</h1>
        <p>Explore our comprehensive API documentation.</p>
    </main>
</body>
</html>`
}
