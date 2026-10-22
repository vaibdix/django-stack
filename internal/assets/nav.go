package assets

import "strings"

func boostOptOut(htmx bool) string {
	if !htmx {
		return ""
	}
	return ` hx-boost="false"`
}

const navLinkClass = "text-body transition-colors duration-300 hover:text-heading"

func authNavLinks(allauth bool, tailwind bool) string {
	if !allauth {
		return djangoAuthNavLinks(tailwind)
	}
	if tailwind {
		return `
                    {% if user.is_authenticated %}
                    <a href="{% url 'account_logout' %}" class="` + navLinkClass + `">Logout</a>
                    {% else %}
                    <a href="{% url 'account_login' %}" class="` + navLinkClass + `">Login</a>
                    <a href="{% url 'account_signup' %}" class="` + navLinkClass + `">Sign Up</a>
                    {% endif %}`
	}
	return `{% if user.is_authenticated %}<a href="{% url 'account_logout' %}">Logout</a>{% else %}<a href="{% url 'account_login' %}">Login</a><a href="{% url 'account_signup' %}">Sign Up</a>{% endif %}`
}

func appNavLink(appName, route string) string {
	if appName == "" {
		return ""
	}
	titleName := appName
	if len(titleName) > 0 {
		titleName = strings.ToUpper(titleName[:1]) + titleName[1:]
	}
	return "\n                    <a href=\"{% url '" + appName + ":" + route + "' %}\" class=\"" + navLinkClass + "\">" + titleName + "</a>"
}

func apiDocsNavLink(showAPIDocs, htmx bool, route string) string {
	if !showAPIDocs {
		return ""
	}
	return "\n                    <a" + boostOptOut(htmx) + " href=\"{% url '" + route + "' %}\" class=\"" + navLinkClass + "\">API Docs</a>"
}

func djangoAuthNavLinks(tailwind bool) string {
	if tailwind {
		return `
                    {% if user.is_authenticated %}
                    <form method="post" action="{% url 'logout' %}" class="m-0 inline">
                        {% csrf_token %}
                        <button type="submit" class="cursor-pointer border-0 bg-transparent p-0 ` + navLinkClass + `">Logout</button>
                    </form>
                    {% else %}
                    <a href="{% url 'login' %}" class="` + navLinkClass + `">Login</a>
                    {% endif %}`
	}
	return `{% if user.is_authenticated %}<form method="post" action="{% url 'logout' %}" class="nav-form">{% csrf_token %}<button type="submit">Logout</button></form>{% else %}<a href="{% url 'login' %}">Login</a>{% endif %}`
}

func plainAppNavLink(appName, route string) string {
	if appName == "" {
		return ""
	}
	return `<a href="{% url '` + appName + `:` + route + `' %}">` + strings.ToUpper(appName[:1]) + appName[1:] + `</a>`
}
