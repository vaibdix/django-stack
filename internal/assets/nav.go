package assets

import "strings"

// Navigation fragments.
//
// The Tailwind variants name tokens declared in the generated project's @theme
// block (see templates/css/tailwind-source.css.tmpl) rather than white-opacity
// literals. A token that is used but never declared produces no CSS and no
// error, so TestEveryTailwindTokenIsDeclared checks the two agree.

const navLinkClass = "text-sm text-body transition hover:text-heading"

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
	return "\n                    <a href=\"/" + appName + "/\" class=\"" + navLinkClass + "\">" + titleName + "</a>"
}

func apiDocsNavLink(showAPIDocs bool) string {
	if !showAPIDocs {
		return ""
	}
	return "\n                    <a href=\"/api/schema/swagger-ui/\" class=\"" + navLinkClass + "\">API Docs</a>"
}
