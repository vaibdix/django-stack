package templates

import "strings"

func authNavLinks(allauth bool, tailwind bool) string {
	if !allauth {
		return ""
	}
	if tailwind {
		return `
                    {% if user.is_authenticated %}
                    <a href="{% url 'account_logout' %}" class="text-sm text-white/60 transition hover:text-white">Logout</a>
                    {% else %}
                    <a href="{% url 'account_login' %}" class="text-sm text-white/60 transition hover:text-white">Login</a>
                    <a href="{% url 'account_signup' %}" class="text-sm text-white/60 transition hover:text-white">Sign Up</a>
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
	return "\n                    <a href=\"/" + appName + "/\" class=\"text-sm text-white/60 transition hover:text-white\">" + titleName + "</a>"
}

func apiDocsNavLink(showAPIDocs bool) string {
	if !showAPIDocs {
		return ""
	}
	return "\n                    <a href=\"/api/schema/swagger-ui/\" class=\"text-sm text-white/60 transition hover:text-white\">API Docs</a>"
}
