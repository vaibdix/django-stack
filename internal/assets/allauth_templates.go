package assets

type authPage struct {
	Name     string
	Title    string
	Heading  string
	Subtitle string
	Action   string
	Submit   string
	Footer   string
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

var AuthNotices = []authPage{
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
	{
		Name:     "email_confirm",
		Title:    "Confirm Email",
		Heading:  "Confirm your email address",
		Subtitle: "Use the button below to confirm that this address belongs to you.",
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

// DjangoLoginTemplate is registration/login.html for django.contrib.auth's
// LoginView, used when the project does not install allauth.
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

	return `{% extends "base.html" %}
{% block title %}` + page.Title + `{% endblock %}
{% block content %}
` + style.open + `
        <h2>` + page.Heading + `</h2>
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
        </form>` + footer + `
` + style.close + `
{% endblock %}`
}

// AuthNoticeTemplate is the allauth page that only tells the user something.
func AuthNoticeTemplate(page authPage, tailwind bool) string {
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

// AuthFormTemplate is the allauth page for a form: sign in, sign up, reset.
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
        <h2 class="auth-title">Sign out</h2>
        <p class="auth-subtitle">Are you sure you want to sign out?</p>
        <form method="post" action="{% url 'account_logout' %}">
            {% csrf_token %}
            {% if redirect_field_value %}<input type="hidden" name="{{ redirect_field_name }}" value="{{ redirect_field_value }}">{% endif %}
            <button type="submit" class="auth-submit">Sign Out</button>
        </form>
    </div>
</div>
{% endblock %}`
}

// authStyle is the markup the auth pages differ by between the Tailwind
// stacks and the plain stylesheet. Choosing it once keeps each page a single
// template instead of a branch per class.
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
	submit:     "auth-submit",
	footer:     "auth-footer",
}

func authStyleFor(tailwind bool) authStyle {
	if tailwind {
		return tailwindAuthStyle
	}
	return plainAuthStyle
}
