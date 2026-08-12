package assets

import "strings"

const navLinkClass = "text-body transition-colors duration-300 hover:text-heading"

func authNavLinks(allauth bool, tailwind bool) string {
	if !allauth {
		return ""
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

func appNavLink(appName string) string {
	if appName == "" {
		return ""
	}
	titleName := appName
	if len(titleName) > 0 {
		titleName = strings.ToUpper(titleName[:1]) + titleName[1:]
	}
	return "\n                    <a href=\"{% url '" + appName + ":book_list' %}\" class=\"" + navLinkClass + "\">" + titleName + "</a>"
}

func apiDocsNavLink(showAPIDocs bool) string {
	if !showAPIDocs {
		return ""
	}
	return "\n                    <a href=\"{% url 'swagger-ui' %}\" class=\"" + navLinkClass + "\">API Docs</a>"
}
