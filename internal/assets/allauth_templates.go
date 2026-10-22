package assets

type authPage struct {
	Name     string
	Title    string
	Heading  string
	Subtitle string
	Action   string
	Submit   string
	Footer   string

	Unless *authCondition
}

type authCondition struct {
	Variable string
	Heading  string
	Subtitle string
	Route    string
	Link     string
}

var AuthPages = []authPage{
	{
		Name:     "login",
		Title:    "Sign In",
		Heading:  "Sign in",
		Subtitle: "Welcome back. Enter your credentials to continue.",
		Action:   "account_login",
		Submit:   "Sign In",
		Footer:   `<a href="{% url 'account_reset_password' %}">Forgot password?</a> · Don't have an account? <a href="{% url 'account_signup' %}">Sign up</a>`,
	},
	{
		Name:     "signup",
		Title:    "Sign Up",
		Heading:  "Create your account",
		Subtitle: "It takes less than a minute.",
		Action:   "account_signup",
		Submit:   "Create Account",
		Footer:   `Already have an account? <a href="{% url 'account_login' %}">Sign in</a>`,
	},
	{
		Name:     "password_reset",
		Title:    "Reset Password",
		Heading:  "Reset your password",
		Subtitle: "We will email you a link to choose a new password.",
		Action:   "account_reset_password",
		Submit:   "Send reset link",
		Footer:   `<a href="{% url 'account_login' %}">Back to sign in</a>`,
	},
	{
		Name:     "password_reset_from_key",
		Title:    "Set a New Password",
		Heading:  "Choose a new password",
		Subtitle: "Pick something you have not used before.",
		Action:   "",
		Submit:   "Save password",
		Footer:   "",
		Unless: &authCondition{
			Variable: "token_fail",
			Heading:  "This link no longer works",
			Subtitle: "The password reset link is invalid, possibly because it has already been used.",
			Route:    "account_reset_password",
			Link:     "Request a new link",
		},
	},
	{
		Name:     "password_change",
		Title:    "Change Password",
		Heading:  "Change your password",
		Subtitle: "Enter your current password, then the new one.",
		Action:   "account_change_password",
		Submit:   "Save password",
		Footer:   "",
	},
}

var authNotices = []authPage{
	{
		Name:     "verification_sent",
		Title:    "Verify Your Email",
		Heading:  "Check your inbox",
		Subtitle: "We sent a verification link to the address you signed up with. Follow it to finish creating your account.",
	},
	{
		Name:     "password_reset_done",
		Title:    "Reset Link Sent",
		Heading:  "Check your inbox",
		Subtitle: "If an account exists for that address, a password reset link is on its way.",
	},
	{
		Name:     "password_reset_from_key_done",
		Title:    "Password Changed",
		Heading:  "Your password is set",
		Subtitle: "You can now sign in with your new password.",
	},
}

const allauthRedirectInput = `{% if redirect_field_value %}<input type="hidden" name="{{ redirect_field_name }}" value="{{ redirect_field_value }}">{% endif %}`

const djangoRedirectInput = `{% if next %}<input type="hidden" name="next" value="{{ next }}">{% endif %}`

var djangoLoginPage = authPage{
	Name:     "login",
	Title:    "Sign In",
	Heading:  "Sign in",
	Subtitle: "Welcome back. Enter your credentials to continue.",
	Action:   "login",
	Submit:   "Sign In",
}

func DjangoLoginTemplate(tailwind bool) string {
	return authFormPageWith(djangoLoginPage, tailwind, djangoRedirectInput)
}

func authFormPageWith(page authPage, tailwind bool, redirectInput string) string {
	style := authStyleFor(tailwind)
	action := ""
	if page.Action != "" {
		action = ` action="{% url '` + page.Action + `' %}"`
	}

	footer := ""
	if page.Footer != "" {
		footer = "\n        <p class=\"" + style.footer + "\">" + page.Footer + "</p>"
	}

	form := `        <h2>` + page.Heading + `</h2>
        <p class="` + style.subtitle + `">` + page.Subtitle + `</p>
        <form method="post"` + action + ` class="` + style.form + `">
            {% csrf_token %}
            {% for error in form.non_field_errors %}<p class="` + style.error + `">{{ error }}</p>{% endfor %}
            {% for field in form %}
            <div class="` + style.field + `">
                {{ field.label_tag }}
                {{ field }}
                {% for error in field.errors %}<p class="` + style.fieldError + `">{{ error }}</p>{% endfor %}
            </div>
            {% endfor %}
            ` + redirectInput + `
            <button type="submit" class="` + style.submit + `">` + page.Submit + `</button>
        </form>` + footer
	if page.Unless != nil {
		form = `        {% if ` + page.Unless.Variable + ` %}
        <h2>` + page.Unless.Heading + `</h2>
        <p class="` + style.subtitle + `">` + page.Unless.Subtitle + `</p>
        <a href="{% url '` + page.Unless.Route + `' %}" class="` + style.submit + `">` + page.Unless.Link + `</a>
        {% else %}
` + form + `
        {% endif %}`
	}

	return `{% extends "base.html" %}
{% block title %}` + page.Title + `{% endblock %}
{% block content %}
` + style.open + `
` + form + `
` + style.close + `
{% endblock %}`
}

func authNoticeTemplate(page authPage, tailwind bool) string {
	style := authStyleFor(tailwind)
	return `{% extends "base.html" %}
{% block title %}` + page.Title + `{% endblock %}
{% block content %}
` + style.open + `
        <h2>` + page.Heading + `</h2>
        <p class="` + style.subtitle + `">` + page.Subtitle + `</p>
        <a href="{% url 'account_login' %}" class="` + style.submit + `">Back to sign in</a>
` + style.close + `
{% endblock %}`
}

func allAuthEmailConfirm(tailwind bool) string {
	style := authStyleFor(tailwind)
	return `{% extends "base.html" %}
{% block title %}Confirm Email{% endblock %}
{% block content %}
` + style.open + `
        <h2>Confirm your email address</h2>
        {% if confirmation %}
        {% if can_confirm %}
        <p class="` + style.subtitle + `">Confirm that {{ confirmation.email_address.email }} is your email address.</p>
        <form method="post" action="{% url 'account_confirm_email' confirmation.key %}" class="` + style.form + `">
            {% csrf_token %}
            ` + allauthRedirectInput + `
            <button type="submit" class="` + style.submit + `">Confirm</button>
        </form>
        {% else %}
        <p class="` + style.subtitle + `">{{ confirmation.email_address.email }} is already confirmed by a different account.</p>
        <a href="{% url 'account_login' %}" class="` + style.submit + `">Back to sign in</a>
        {% endif %}
        {% else %}
        <p class="` + style.subtitle + `">This confirmation link has expired or is invalid. Sign in and a new one is sent to you.</p>
        <a href="{% url 'account_login' %}" class="` + style.submit + `">Back to sign in</a>
        {% endif %}
` + style.close + `
{% endblock %}`
}

func AllAuthTemplates(tailwind bool) map[string]string {
	pages := map[string]string{
		"logout.html":        AllAuthLogout(tailwind),
		"email_confirm.html": allAuthEmailConfirm(tailwind),
	}
	for _, page := range AuthPages {
		pages[page.Name+".html"] = AuthFormTemplate(page, tailwind)
	}
	for _, page := range authNotices {
		pages[page.Name+".html"] = authNoticeTemplate(page, tailwind)
	}
	return pages
}

func AuthFormTemplate(page authPage, tailwind bool) string {
	return authFormPageWith(page, tailwind, allauthRedirectInput)
}

func AllAuthLogout(tailwind bool) string {
	if tailwind {
		return `{% extends "base.html" %}
{% block title %}Sign Out{% endblock %}
{% block content %}
<section class="flex min-h-[50vh] items-center justify-center px-5 py-8">
    <div class="panel w-full max-w-[420px] p-6 text-center">
        <h2>Sign out</h2>
        <p class="mb-6 text-body">Are you sure you want to sign out?</p>
        <form method="post" action="{% url 'account_logout' %}">
            {% csrf_token %}
            {% if redirect_field_value %}<input type="hidden" name="{{ redirect_field_name }}" value="{{ redirect_field_value }}">{% endif %}
            <button type="submit" class="button button-primary w-full">Sign Out</button>
        </form>
    </div>
</section>
{% endblock %}`
	}
	return `{% extends "base.html" %}
{% block title %}Sign Out{% endblock %}
{% block content %}
<div class="auth-container">
    <div class="auth-card auth-card--center">
        <h2>Sign out</h2>
        <p class="auth-subtitle">Are you sure you want to sign out?</p>
        <form method="post" action="{% url 'account_logout' %}">
            {% csrf_token %}
            {% if redirect_field_value %}<input type="hidden" name="{{ redirect_field_name }}" value="{{ redirect_field_value }}">{% endif %}
            <button type="submit" class="` + plainAuthStyle.submit + `">Sign Out</button>
        </form>
    </div>
</div>
{% endblock %}`
}

type authStyle struct {
	open, close                                              string
	subtitle, form, field, error, fieldError, submit, footer string
}

var tailwindAuthStyle = authStyle{
	open: `<section class="flex min-h-[50vh] items-center justify-center px-5 py-8">
    <div class="panel w-full max-w-[420px] p-6">`,
	close: `    </div>
</section>`,
	subtitle:   "mb-6 text-body",
	form:       "space-y-4",
	field:      "space-y-1",
	error:      "rounded-[6px] border border-danger px-3 py-2 text-danger",
	fieldError: "mt-1 text-[14px] text-danger",
	submit:     "button button-primary w-full",
	footer:     "mt-6 text-center text-body",
}

var plainAuthStyle = authStyle{
	open: `<div class="auth-container">
    <div class="auth-card">`,
	close: `    </div>
</div>`,
	subtitle:   "auth-subtitle",
	form:       "auth-form",
	field:      "auth-field",
	error:      "auth-error",
	fieldError: "auth-field-error",
	submit:     "button button-primary auth-submit",
	footer:     "auth-footer",
}

func authStyleFor(tailwind bool) authStyle {
	if tailwind {
		return tailwindAuthStyle
	}
	return plainAuthStyle
}
