package templates

import (
	"django-stack/internal/template"
)

var (
	basicBaseTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{% block title %}{{ project_name|default:"Django Site" }}{% endblock %}</title>
    <style>
        :root {
            --color-white: #ffffff;
            --color-black: #000000;
            --color-gray-300: #d1d5db;
            --color-gray-400: #9ca3af;
            --color-gray-700: #374151;
            --color-gray-800: #1f2937;
            --color-gray-900: #111827;
        }
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            background-color: var(--color-black);
            color: var(--color-white);
            line-height: 1.5;
            min-height: 100vh;
            display: flex;
            flex-direction: column;
        }
        header { position: sticky; top: 0; z-index: 50; background-color: rgba(0,0,0,0.8); border-bottom: 1px solid var(--color-gray-800); }
        .container { max-width: 1280px; margin: 0 auto; padding: 0 1rem; }
        .header-content { display: flex; justify-content: space-between; align-items: center; padding: 1rem 0; }
        .logo { display: flex; align-items: center; gap: 0.5rem; text-decoration: none; color: var(--color-white); }
        .logo h1 { font-size: 1.25rem; font-weight: 600; }
        nav { display: none; }
        @media (min-width: 768px) { nav { display: flex; gap: 2rem; } }
        nav a { color: var(--color-gray-300); text-decoration: none; font-size: 0.875rem; }
        nav a:hover { color: var(--color-white); }
        main { flex: 1; }
        footer { border-top: 1px solid var(--color-gray-800); margin-top: 5rem; padding: 4rem 0; }
        .footer-content { display: grid; grid-template-columns: 1fr; gap: 2rem; }
        @media (min-width: 768px) { .footer-content { grid-template-columns: 2fr 1fr; } }
        .footer-brand { display: flex; align-items: center; gap: 0.5rem; margin-bottom: 1rem; }
        .footer-brand span { font-size: 1.125rem; font-weight: 600; }
        .footer-description { color: var(--color-gray-400); max-width: 24rem; margin-bottom: 1.5rem; }
        .footer-links h3 { color: var(--color-white); font-size: 0.875rem; font-weight: 600; margin-bottom: 1rem; }
        .footer-links ul { list-style: none; }
        .footer-links li:not(:last-child) { margin-bottom: 0.75rem; }
        .footer-links a { color: var(--color-gray-400); text-decoration: none; font-size: 0.875rem; }
        .footer-links a:hover { color: var(--color-white); }
        .footer-bottom { border-top: 1px solid var(--color-gray-800); margin-top: 3rem; padding-top: 2rem; display: flex; justify-content: space-between; }
    </style>
    {% block extra_head %}{% endblock %}
</head>
<body>
    <header>
        <div class="container">
            <div class="header-content">
                <a href="/" class="logo">
                    <svg width="32" height="32" viewBox="0 0 32 32" fill="none">
                        <rect width="32" height="32" rx="8" fill="white"/>
                        <path d="M12 8L20 16L12 24" stroke="black" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
                    </svg>
                    <h1>{{ project_name|default:"Django" }}</h1>
                </a>
                <nav><a href="/">Home</a><a href="/admin/">Admin</a>{% if app_name %}<a href="/{{ app_name }}/">{{ app_name|title }}</a>{% endif %}</nav>
            </div>
        </div>
    </header>
    <main>{% block content %}{% endblock %}</main>
    <footer>
        <div class="container">
            <div class="footer-content">
                <div>
                    <div class="footer-brand"><span>{{ project_name|default:"Django Site" }}</span></div>
                    <p class="footer-description">The Django framework that gives you everything you need.</p>
                </div>
                <div class="footer-links">
                    <h3>Resources</h3>
                    <ul><li><a href="/admin/">Admin Panel</a></li><li><a href="https://docs.djangoproject.com/">Django Docs</a></li></ul>
                </div>
            </div>
            <div class="footer-bottom">
                <p>© {% now "Y" %} {{ project_name|default:"Django Site" }}. All rights reserved.</p>
            </div>
        </div>
    </footer>
    {% block extra_body %}{% endblock %}
</body>
</html>`

	basicIndexTemplate = `{% extends 'base.html' %}

{% block title %}{{ project_name }} - Home{% endblock %}

{% block extra_head %}
<style>
    .hero { position: relative; overflow: hidden; min-height: 80vh; display: flex; align-items: center; justify-content: center; text-align: center; padding: 4rem 1rem; }
    .hero-gradient { position: absolute; inset: 0; background: linear-gradient(to bottom, transparent, var(--color-black)); pointer-events: none; }
    .hero-content { position: relative; z-index: 1; }
    .badge { display: inline-flex; align-items: center; border: 1px solid var(--color-gray-800); background-color: rgba(17,24,39,0.5); padding: 0.5rem 1rem; border-radius: 9999px; font-size: 0.875rem; color: var(--color-gray-300); margin-bottom: 2rem; }
    .hero-title { font-size: 3rem; font-weight: 700; line-height: 1.2; margin-bottom: 2rem; }
    @media (min-width: 768px) { .hero-title { font-size: 4rem; } }
    .gradient-text { background: linear-gradient(to right, #60a5fa, #a78bfa, #f472b6); -webkit-background-clip: text; background-clip: text; color: transparent; }
    .hero-subtitle { font-size: 1.25rem; color: var(--color-gray-400); max-width: 48rem; margin: 0 auto 3rem; }
    .button-group { display: flex; gap: 1rem; justify-content: center; flex-wrap: wrap; }
    .button { display: inline-flex; align-items: center; padding: 1rem 2rem; border-radius: 0.375rem; font-weight: 600; transition: all 0.2s; text-decoration: none; }
    .button-primary { background-color: var(--color-white); color: var(--color-black); }
    .button-primary:hover { background-color: var(--color-gray-300); }
    .button-secondary { border: 1px solid var(--color-gray-700); color: var(--color-white); }
    .button-secondary:hover { border-color: var(--color-gray-600); }
    .features { max-width: 1280px; margin: 0 auto; padding: 6rem 1rem; }
    .features-header { text-align: center; margin-bottom: 4rem; }
    .features-title { font-size: 1.875rem; font-weight: 700; color: var(--color-white); margin-bottom: 1rem; }
    .features-subtitle { font-size: 1.25rem; color: var(--color-gray-400); }
    .features-grid { display: grid; grid-template-columns: 1fr; gap: 2rem; }
    @media (min-width: 768px) { .features-grid { grid-template-columns: repeat(2, 1fr); } }
    .feature-card { border: 1px solid var(--color-gray-800); background-color: var(--color-gray-900); border-radius: 0.5rem; padding: 2rem; }
    .feature-title { font-size: 1.25rem; font-weight: 600; color: var(--color-white); margin-bottom: 0.75rem; }
    .feature-description { color: var(--color-gray-400); }
</style>
{% endblock %}

{% block content %}
<div class="hero">
    <div class="hero-gradient"></div>
    <div class="hero-content">
        <div class="badge"><span>Production ready Django application</span></div>
        <h1 class="hero-title"><span>The Django</span><span class="gradient-text">Framework</span></h1>
        <p class="hero-subtitle">Django provides everything you need to build fast, secure, and scalable web applications.</p>
        <div class="button-group">
            <a href="https://docs.djangoproject.com/" class="button button-primary">Get Started</a>
            <a href="/admin/" class="button button-secondary">Admin Panel</a>
        </div>
    </div>
</div>
<div class="features">
    <div class="features-header">
        <h2 class="features-title">Why Django?</h2>
        <p class="features-subtitle">Built for speed, security, and scalability.</p>
    </div>
    <div class="features-grid">
        <div class="feature-card">
            <h3 class="feature-title">Fast Development</h3>
            <p class="feature-description">Batteries-included approach means you can build full-featured applications quickly.</p>
        </div>
        <div class="feature-card">
            <h3 class="feature-title">Security First</h3>
            <p class="feature-description">Built-in protection against common security threats.</p>
        </div>
    </div>
</div>
{% endblock %}`
)

func TemplateContent(templateType string, config map[string]string) string {
	apiLink := ""
	if config["api"] == "true" {
		apiLink = `<a href="/api/schema/swagger-ui/">API Docs</a>`
	}
	allauthEnabled := config["allauth"] == "true"
	authLinks := authNavLinks(allauthEnabled, config["tailwind"] == "true")
	htmxEnabled := config["htmx"] == "true"
	htmxLoadTag := ""
	htmxScriptTag := ""
	htmxBodyAttr := ""
	if htmxEnabled {
		htmxLoadTag = "\n    {% load django_htmx %}"
		htmxScriptTag = "\n    {% htmx_script %}"
		htmxBodyAttr = ` hx-headers='{"X-CSRFToken": "{{ csrf_token }}"}'`
	}
	switch templateType {
	case "base.html":
		if config["tailwind"] == "true" {
			return tailwindBaseTemplateContent(config["app_name"], config["api"] == "true", htmxEnabled, allauthEnabled)
		}
		if config["global_templates"] == "true" {
			return `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{% block title %}{{ project_name|default:"Django Site" }}{% endblock %}</title>
    {% load static %}` + htmxLoadTag + `
    <link rel="stylesheet" href="{% static 'css/base.css' %}">` + htmxScriptTag + `
    {% block extra_head %}{% endblock %}
</head>
<body` + htmxBodyAttr + `>
    <header>
        <div class="container">
            <div class="header-content">
                <a href="/" class="logo">
                    <svg width="32" height="32" viewBox="0 0 32 32" fill="none">
                        <rect width="32" height="32" rx="8" fill="white"/>
                        <path d="M12 8L20 16L12 24" stroke="black" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
                    </svg>
                    <h1>{{ project_name|default:"Django" }}</h1>
                </a>
                <nav><a href="/">Home</a><a href="/admin/">Admin</a>{% if app_name %}<a href="/{{ app_name }}/">{{ app_name|title }}</a>{% endif %}` + apiLink + authLinks + `</nav>
            </div>
        </div>
    </header>
    <main>{% block content %}{% endblock %}</main>
    <footer>
        <div class="container">
            <div class="footer-bottom"><p>© {% now "Y" %} {{ project_name|default:"Django Site" }}. All rights reserved.</p></div>
        </div>
    </footer>
    {% block extra_body %}{% endblock %}
</body>
</html>`
		}
		return `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{% block title %}{{ project_name|default:"Django Site" }}{% endblock %}</title>` + htmxLoadTag + `
    <style>` + template.GetBaseCSS() + `</style>` + htmxScriptTag + `
    {% block extra_head %}{% endblock %}
</head>
<body` + htmxBodyAttr + `>
    <header>
        <div class="container">
            <div class="header-content">
                <a href="/" class="logo">
                    <svg width="32" height="32" viewBox="0 0 32 32" fill="none">
                        <rect width="32" height="32" rx="8" fill="white"/>
                        <path d="M12 8L20 16L12 24" stroke="black" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
                    </svg>
                    <h1>{{ project_name|default:"Django" }}</h1>
                </a>
                <nav><a href="/">Home</a><a href="/admin/">Admin</a>{% if app_name %}<a href="/{{ app_name }}/">{{ app_name|title }}</a>{% endif %}` + apiLink + authLinks + `</nav>
            </div>
        </div>
    </header>
    <main>{% block content %}{% endblock %}</main>
    <footer>
        <div class="container">
            <div class="footer-bottom"><p>© {% now "Y" %} {{ project_name|default:"Django Site" }}. All rights reserved.</p></div>
        </div>
    </footer>
    {% block extra_body %}{% endblock %}
</body>
</html>`

	case "index.html":
		if config["tailwind"] == "true" {
			return tailwindIndexTemplateContent()
		}
		if config["global_templates"] == "true" {
			return `{% extends 'base.html' %}
{% load static %}
{% block title %}{{ project_name }} - Home{% endblock %}
{% block extra_head %}
<link rel="stylesheet" href="{% static 'css/home.css' %}">
{% endblock %}
{% block content %}
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
		return `{% extends 'base.html' %}
{% block title %}{{ project_name }} - Home{% endblock %}
{% block extra_head %}
<style>` + template.GetHomeCSS() + `</style>
{% endblock %}
{% block content %}
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

	case "api-docs.html":
		if config["tailwind"] == "true" {
			return tailwindApiDocsTemplateContent()
		}
		return `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{ project_name }} - API Documentation</title>
    <style>` + template.GetBaseCSS() + `</style>
</head>
<body>
    <header>
        <div class="container">
            <div class="header-content">
                <a href="/" class="logo"><h1>{{ project_name }}</h1></a>
                <nav><a href="/">Home</a><a href="/admin/">Admin</a>{% if app_name %}<a href="/{{ app_name }}/">{{ app_name|title }}</a>{% endif %}` + apiLink + `</nav>
            </div>
        </div>
    </header>
    <main class="container" style="padding: 4rem 1rem;">
        <h1>API Documentation</h1>
        <p>Explore our comprehensive API documentation.</p>
    </main>
</body>
</html>`

	default:
		return ""
	}
}
