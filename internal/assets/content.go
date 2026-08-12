package assets

import "strings"

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

func apiDocsLink(enabled, htmx bool, route string) string {
	if !enabled {
		return ""
	}
	return `<a` + boostOptOut(htmx) + ` href="{% url '` + route + `' %}">API Docs</a>`
}

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
		BodyAttr: ` hx-boost="true" hx-headers='{"X-CSRFToken": "{{ csrf_token }}"}'`,
	}
}

func sharedNav(opts PageOptions) string {
	return `<nav><a href="{% url 'home' %}">Home</a>` +
		`{% if app_name %}<a href="{% url app_name|add:':book_list' %}">{{ app_name|title }}</a>{% endif %}` +
		apiDocsLink(opts.API, opts.HTMX, opts.docsRoute()) +
		`<a` + boostOptOut(opts.HTMX) + ` href="{% url 'admin:index' %}">Admin</a>` +
		authNavLinks(opts.AllAuth, opts.Tailwind) + `</nav>`
}

func pageHeader() string {
	return `  <div class="frame">
    <header>
        <div class="container">
            <div class="header-content">
                <a href="{% url 'home' %}" class="logo">
                    <svg width="24" height="24" viewBox="0 0 32 32" fill="none" aria-hidden="true">
                        <rect width="32" height="32" rx="6" fill="currentColor" opacity="0.12"/>
                        <path d="M12 8L20 16L12 24" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"/>
                    </svg>
                    <h1>{{ project_name|default:"Django" }}</h1>
                </a>
                `
}

func pageFooter(htmx bool) string {
	return `    <main>{% block content %}{% endblock %}</main>
    <section id="next-steps" class="ticks">
        <div id="docs">
            <svg class="icon" viewBox="0 0 22 22" fill="none" stroke="currentColor" stroke-width="1.6" aria-hidden="true"><path d="M4 3h9l5 5v11H4z"/><path d="M13 3v5h5"/></svg>
            <h2>Documentation</h2>
            <p>Your questions, answered</p>
            <ul>
                <li><a` + boostOptOut(htmx) + ` href="https://docs.djangoproject.com/">Explore Django</a></li>
                <li><a` + boostOptOut(htmx) + ` href="{% url 'admin:index' %}">Open admin</a></li>
            </ul>
        </div>
        <div id="build">
            <svg class="icon" viewBox="0 0 22 22" fill="none" stroke="currentColor" stroke-width="1.6" aria-hidden="true"><path d="M11 2l7 4v10l-7 4-7-4V6z"/><path d="M11 11l7-4M11 11v9M11 11L4 7"/></svg>
            <h2>Start building</h2>
            <p>Register a model and it appears in the admin</p>
            <ul>
                <li><a` + boostOptOut(htmx) + ` href="https://docs.djangoproject.com/en/stable/intro/tutorial01/">Tutorial</a></li>
                <li><a` + boostOptOut(htmx) + ` href="https://docs.djangoproject.com/en/stable/ref/models/">Model reference</a></li>
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
		return tailwindBaseTemplateContent(opts)
	}

	htmx := htmxTagsFor(opts.HTMX)

	var styles string
	if opts.GlobalTemplates {
		styles = `
    {% load static %}` + htmx.Load + `
    <link rel="stylesheet" href="{% static 'css/base.css' %}">
    <link rel="stylesheet" href="{% static 'css/home.css' %}">
    <link rel="stylesheet" href="{% static 'css/app.css' %}">
    <link rel="stylesheet" href="{% static 'css/auth.css' %}">` + htmx.Script
	} else {
		styles = htmx.Load + `
    <style>` + baseCSS + homeCSS + appCSS + authCSS + `</style>` + htmx.Script
	}

	var b strings.Builder
	b.WriteString(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <meta name="description" content="{{ project_name }} — a Django application.">
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
	b.WriteString(pageFooter(opts.HTMX))
	return b.String()
}

func heroAndFeatures(htmx bool) string {
	return `{% block content %}
<section id="center">
    <div>
        <h1>Get started</h1>
        <p class="hero-subtitle">Edit <code>{{ project_name|default:"project" }}/views.py</code> and save to see your changes</p>
    </div>
    <span class="badge">Django is running</span>
    <div class="button-group">
        <a` + boostOptOut(htmx) + ` href="https://docs.djangoproject.com/" class="button button-primary">Read the docs</a>
        <a` + boostOptOut(htmx) + ` href="{% url 'admin:index' %}" class="button button-secondary">Open admin</a>
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
		return tailwindIndexTemplateContent(opts.HTMX)
	}

	return `{% extends 'base.html' %}
{% block title %}{{ project_name }} - Home{% endblock %}
` + heroAndFeatures(opts.HTMX)
}

func renderAPIDocsPage(opts PageOptions) string {
	if opts.Tailwind {
		return tailwindAPIDocsTemplateContent()
	}

	return `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <meta name="description" content="{{ project_name }} — a Django application.">
    <title>{{ project_name }} - API Documentation</title>
    <style>` + baseCSS + `</style>
</head>
<body>
  <div class="frame">
    <header>
        <div class="container">
            <div class="header-content">
                <a href="{% url 'home' %}" class="logo"><h1>{{ project_name }}</h1></a>
                <nav><a href="{% url 'home' %}">Home</a><a` + boostOptOut(opts.HTMX) + ` href="{% url 'admin:index' %}">Admin</a>{% if app_name %}<a href="{% url app_name|add:':book_list' %}">{{ app_name|title }}</a>{% endif %}` + apiDocsLink(opts.API, opts.HTMX, opts.docsRoute()) + `</nav>
            </div>
        </div>
    </header>
    <main class="container">
        <h1>API Documentation</h1>
        <p>Explore the browsable API, the generated schema and the admin from one place.</p>
        <p>Anonymous callers get read access only. Sign in at <code>/api-auth/login/</code> and the
           create, edit and delete forms appear on every endpoint.</p>
    </main>
  </div>
</body>
</html>`
}
