package assets

// The django-allauth pages.
//
// The field names and ids here are a contract with allauth, not a styling
// choice: it binds to `login`, `password`, `username`, `email`, `password1` and
// `password2`, and renders errors through form.<field>.errors. A class-list
// edit that renames one produces a form that submits nothing allauth
// recognises, with no error at generation time — which is why
// TestAllAuthPagesKeepTheirFormContract asserts each of them.

// The shared shape of an auth card. Both variants centre a single bordered
// panel; only the mechanism differs — utility classes against a stylesheet.
const (
	authInputClass = "mt-2 w-full rounded-xl border border-rule bg-surface px-4 py-3 text-sm text-heading " +
		"placeholder:text-body outline-none transition focus:border-accent"
	authLabelClass  = "block text-xs uppercase tracking-[0.22em] text-body"
	authSubmitClass = "btn btn-primary w-full"
	authErrorClass  = "rounded-xl border border-rule bg-surface px-4 py-3 text-sm text-accent"
	authFieldError  = "mt-1 text-xs text-accent"
)

// authStyles is the plain-CSS equivalent of the classes above, inlined into the
// page's extra_head block. It names the same tokens base.css declares, so the
// two variants stay the same design.
const authStyles = `<style>
    .auth-container { display: flex; min-height: 70vh; align-items: center; justify-content: center; padding: 2rem 1.5rem; }
    .auth-card { width: 100%; max-width: 26rem; border: 1px solid var(--rule); background: var(--surface); border-radius: 16px; padding: 2rem; }
    .auth-label { display: block; font-size: 0.6875rem; text-transform: uppercase; letter-spacing: 0.22em; color: var(--body); }
    .auth-title { margin-top: 0.75rem; font-size: 1.5rem; font-weight: 600; letter-spacing: -0.03em; color: var(--heading); }
    .auth-subtitle { margin-top: 0.5rem; font-size: 0.875rem; color: var(--body); }
    .auth-form { margin-top: 1.75rem; }
    .auth-form > * + * { margin-top: 1.125rem; }
    .auth-input { margin-top: 0.5rem; width: 100%; border: 1px solid var(--rule); background: var(--ground); border-radius: 12px; padding: 0.7rem 0.9rem; font-size: 0.875rem; color: var(--heading); outline: none; }
    .auth-input:focus { border-color: var(--accent); }
    .auth-input::placeholder { color: var(--body); }
    .auth-btn { margin-top: 1.25rem; width: 100%; background: var(--accent-bg); color: var(--accent); border: 1px solid var(--accent); border-radius: 9999px; padding: 0.7rem 1.35rem; font-size: 0.875rem; font-weight: 600; cursor: pointer; }
    .auth-btn:hover { background: color-mix(in srgb, var(--accent) 22%, var(--ground)); }
    .auth-footer { margin-top: 1.5rem; text-align: center; font-size: 0.875rem; color: var(--body); }
    .auth-footer a { color: var(--heading); text-decoration: none; }
    .auth-footer a:hover { color: var(--accent); }
    .auth-error { border: 1px solid var(--rule); background: var(--ground); border-radius: 12px; padding: 0.7rem 0.9rem; font-size: 0.875rem; color: var(--accent); }
    .field-error { margin-top: 0.25rem; font-size: 0.75rem; color: var(--accent); }
</style>`

// AllAuthLogin returns the django-allauth login page, styled for Tailwind or
// plain CSS.
func AllAuthLogin(tailwind bool) string {
	if tailwind {
		return `{% extends 'base.html' %}
{% block title %}Sign In{% endblock %}
{% block content %}
<section class="flex min-h-[70vh] items-center justify-center px-6 py-10">
    <div class="w-full max-w-md rounded-2xl border border-rule bg-surface p-8">
        <p class="text-xs uppercase tracking-[0.22em] text-body">Account</p>
        <h1 class="mt-3 text-2xl font-semibold tracking-[-0.03em] text-heading">Sign in</h1>
        <p class="mt-2 text-sm text-body">Welcome back. Enter your credentials to continue.</p>
        <form method="post" action="{% url 'account_login' %}" class="mt-7 space-y-5">
            {% csrf_token %}
            {% if form.non_field_errors %}
            <div class="` + authErrorClass + `">
                {% for error in form.non_field_errors %}{{ error }}{% endfor %}
            </div>
            {% endif %}
            <div>
                <label for="id_login" class="` + authLabelClass + `">Username or Email</label>
                <input type="text" name="login" id="id_login" value="{{ form.login.value|default:'' }}" class="` + authInputClass + `" placeholder="you@example.com" required>
                {% if form.login.errors %}<p class="` + authFieldError + `">{{ form.login.errors.0 }}</p>{% endif %}
            </div>
            <div>
                <label for="id_password" class="` + authLabelClass + `">Password</label>
                <input type="password" name="password" id="id_password" class="` + authInputClass + `" placeholder="••••••••" required>
                {% if form.password.errors %}<p class="` + authFieldError + `">{{ form.password.errors.0 }}</p>{% endif %}
            </div>
            {% if redirect_field_value %}<input type="hidden" name="{{ redirect_field_name }}" value="{{ redirect_field_value }}">{% endif %}
            <button type="submit" class="` + authSubmitClass + `">Sign In</button>
        </form>
        <p class="mt-6 text-center text-sm text-body">Don't have an account? <a href="{% url 'account_signup' %}" class="text-heading transition hover:text-accent">Sign up</a></p>
    </div>
</section>
{% endblock %}`
	}
	return `{% extends 'base.html' %}
{% block title %}Sign In{% endblock %}
{% block extra_head %}
` + authStyles + `
{% endblock %}
{% block content %}
<div class="auth-container">
    <div class="auth-card">
        <span class="auth-label">Account</span>
        <h1 class="auth-title">Sign in</h1>
        <p class="auth-subtitle">Welcome back. Enter your credentials to continue.</p>
        <form method="post" action="{% url 'account_login' %}" class="auth-form">
            {% csrf_token %}
            {% if form.non_field_errors %}
            <div class="auth-error">{% for error in form.non_field_errors %}{{ error }}{% endfor %}</div>
            {% endif %}
            <div>
                <label for="id_login" class="auth-label">Username or Email</label>
                <input type="text" name="login" id="id_login" value="{{ form.login.value|default:'' }}" class="auth-input" placeholder="you@example.com" required>
                {% if form.login.errors %}<p class="field-error">{{ form.login.errors.0 }}</p>{% endif %}
            </div>
            <div>
                <label for="id_password" class="auth-label">Password</label>
                <input type="password" name="password" id="id_password" class="auth-input" placeholder="••••••••" required>
                {% if form.password.errors %}<p class="field-error">{{ form.password.errors.0 }}</p>{% endif %}
            </div>
            {% if redirect_field_value %}<input type="hidden" name="{{ redirect_field_name }}" value="{{ redirect_field_value }}">{% endif %}
            <button type="submit" class="auth-btn">Sign In</button>
        </form>
        <p class="auth-footer">Don't have an account? <a href="{% url 'account_signup' %}">Sign up</a></p>
    </div>
</div>
{% endblock %}`
}

// AllAuthSignup returns the django-allauth signup page.
func AllAuthSignup(tailwind bool) string {
	if tailwind {
		return `{% extends 'base.html' %}
{% block title %}Sign Up{% endblock %}
{% block content %}
<section class="flex min-h-[70vh] items-center justify-center px-6 py-10">
    <div class="w-full max-w-md rounded-2xl border border-rule bg-surface p-8">
        <p class="text-xs uppercase tracking-[0.22em] text-body">Get started</p>
        <h1 class="mt-3 text-2xl font-semibold tracking-[-0.03em] text-heading">Create an account</h1>
        <p class="mt-2 text-sm text-body">Fill in the details below to get started.</p>
        <form method="post" action="{% url 'account_signup' %}" class="mt-7 space-y-5">
            {% csrf_token %}
            {% if form.non_field_errors %}
            <div class="` + authErrorClass + `">
                {% for error in form.non_field_errors %}{{ error }}{% endfor %}
            </div>
            {% endif %}
            <div>
                <label for="id_username" class="` + authLabelClass + `">Username</label>
                <input type="text" name="username" id="id_username" value="{{ form.username.value|default:'' }}" class="` + authInputClass + `" placeholder="johndoe" required>
                {% if form.username.errors %}<p class="` + authFieldError + `">{{ form.username.errors.0 }}</p>{% endif %}
            </div>
            <div>
                <label for="id_email" class="` + authLabelClass + `">Email</label>
                <input type="email" name="email" id="id_email" value="{{ form.email.value|default:'' }}" class="` + authInputClass + `" placeholder="you@example.com" required>
                {% if form.email.errors %}<p class="` + authFieldError + `">{{ form.email.errors.0 }}</p>{% endif %}
            </div>
            <div>
                <label for="id_password1" class="` + authLabelClass + `">Password</label>
                <input type="password" name="password1" id="id_password1" class="` + authInputClass + `" placeholder="••••••••" required>
                {% if form.password1.errors %}<p class="` + authFieldError + `">{{ form.password1.errors.0 }}</p>{% endif %}
            </div>
            <div>
                <label for="id_password2" class="` + authLabelClass + `">Confirm Password</label>
                <input type="password" name="password2" id="id_password2" class="` + authInputClass + `" placeholder="••••••••" required>
                {% if form.password2.errors %}<p class="` + authFieldError + `">{{ form.password2.errors.0 }}</p>{% endif %}
            </div>
            {% if redirect_field_value %}<input type="hidden" name="{{ redirect_field_name }}" value="{{ redirect_field_value }}">{% endif %}
            <button type="submit" class="` + authSubmitClass + `">Create Account</button>
        </form>
        <p class="mt-6 text-center text-sm text-body">Already have an account? <a href="{% url 'account_login' %}" class="text-heading transition hover:text-accent">Sign in</a></p>
    </div>
</section>
{% endblock %}`
	}
	return `{% extends 'base.html' %}
{% block title %}Sign Up{% endblock %}
{% block extra_head %}
` + authStyles + `
{% endblock %}
{% block content %}
<div class="auth-container">
    <div class="auth-card">
        <span class="auth-label">Get started</span>
        <h1 class="auth-title">Create an account</h1>
        <p class="auth-subtitle">Fill in the details below to get started.</p>
        <form method="post" action="{% url 'account_signup' %}" class="auth-form">
            {% csrf_token %}
            {% if form.non_field_errors %}
            <div class="auth-error">{% for error in form.non_field_errors %}{{ error }}{% endfor %}</div>
            {% endif %}
            <div>
                <label for="id_username" class="auth-label">Username</label>
                <input type="text" name="username" id="id_username" value="{{ form.username.value|default:'' }}" class="auth-input" placeholder="johndoe" required>
                {% if form.username.errors %}<p class="field-error">{{ form.username.errors.0 }}</p>{% endif %}
            </div>
            <div>
                <label for="id_email" class="auth-label">Email</label>
                <input type="email" name="email" id="id_email" value="{{ form.email.value|default:'' }}" class="auth-input" placeholder="you@example.com" required>
                {% if form.email.errors %}<p class="field-error">{{ form.email.errors.0 }}</p>{% endif %}
            </div>
            <div>
                <label for="id_password1" class="auth-label">Password</label>
                <input type="password" name="password1" id="id_password1" class="auth-input" placeholder="••••••••" required>
                {% if form.password1.errors %}<p class="field-error">{{ form.password1.errors.0 }}</p>{% endif %}
            </div>
            <div>
                <label for="id_password2" class="auth-label">Confirm Password</label>
                <input type="password" name="password2" id="id_password2" class="auth-input" placeholder="••••••••" required>
                {% if form.password2.errors %}<p class="field-error">{{ form.password2.errors.0 }}</p>{% endif %}
            </div>
            {% if redirect_field_value %}<input type="hidden" name="{{ redirect_field_name }}" value="{{ redirect_field_value }}">{% endif %}
            <button type="submit" class="auth-btn">Create Account</button>
        </form>
        <p class="auth-footer">Already have an account? <a href="{% url 'account_login' %}">Sign in</a></p>
    </div>
</div>
{% endblock %}`
}

// AllAuthLogout returns the django-allauth logout confirmation page.
func AllAuthLogout(tailwind bool) string {
	if tailwind {
		return `{% extends 'base.html' %}
{% block title %}Sign Out{% endblock %}
{% block content %}
<section class="flex min-h-[70vh] items-center justify-center px-6 py-10">
    <div class="w-full max-w-md rounded-2xl border border-rule bg-surface p-8 text-center">
        <p class="text-xs uppercase tracking-[0.22em] text-body">Account</p>
        <h1 class="mt-3 text-2xl font-semibold tracking-[-0.03em] text-heading">Sign out</h1>
        <p class="mt-2 text-sm text-body">Are you sure you want to sign out?</p>
        <form method="post" action="{% url 'account_logout' %}" class="mt-7">
            {% csrf_token %}
            <button type="submit" class="` + authSubmitClass + `">Sign Out</button>
        </form>
        <p class="mt-6 text-sm text-body"><a href="/" class="text-heading transition hover:text-accent">Go back home</a></p>
    </div>
</section>
{% endblock %}`
	}
	return `{% extends 'base.html' %}
{% block title %}Sign Out{% endblock %}
{% block extra_head %}
` + authStyles + `
{% endblock %}
{% block content %}
<div class="auth-container">
    <div class="auth-card" style="text-align: center;">
        <span class="auth-label">Account</span>
        <h1 class="auth-title">Sign out</h1>
        <p class="auth-subtitle">Are you sure you want to sign out?</p>
        <form method="post" action="{% url 'account_logout' %}">
            {% csrf_token %}
            <button type="submit" class="auth-btn">Sign Out</button>
        </form>
        <p class="auth-footer"><a href="/">Go back home</a></p>
    </div>
</div>
{% endblock %}`
}
