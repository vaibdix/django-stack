package assets

const (
	authInputClass = "mt-2 w-full rounded-[6px] border border-rule bg-ground px-3 py-1.5 " +
		"text-heading outline-none transition-colors duration-300 focus:border-accent-border"
	authLabelClass  = "block text-body"
	authSubmitClass = "button button-primary w-full"
	authErrorClass  = "rounded-[6px] border border-rule bg-surface px-3 py-2 text-accent"
	authFieldError  = "mt-1 text-[14px] text-accent"
)

func AllAuthLogin(tailwind bool) string {
	if tailwind {
		return `{% extends 'base.html' %}
{% block title %}Sign In{% endblock %}
{% block content %}
<section class="flex min-h-[50vh] items-center justify-center px-5 py-8">
    <div class="panel w-full max-w-[420px] p-6">
        <h2>Sign in</h2>
        <p class="mb-6 text-body">Welcome back. Enter your credentials to continue.</p>
        <form method="post" action="{% url 'account_login' %}" class="space-y-4">
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
        <p class="mt-6 text-center text-body">Don't have an account? <a href="{% url 'account_signup' %}" class="text-heading no-underline hover:text-accent">Sign up</a></p>
    </div>
</section>
{% endblock %}`
	}
	return `{% extends 'base.html' %}
{% block title %}Sign In{% endblock %}
{% block content %}
<div class="auth-container">
    <div class="auth-card">
        <h2 class="auth-title">Sign in</h2>
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
            <button type="submit" class="button button-primary auth-btn">Sign In</button>
        </form>
        <p class="auth-footer">Don't have an account? <a href="{% url 'account_signup' %}">Sign up</a></p>
    </div>
</div>
{% endblock %}`
}

func AllAuthSignup(tailwind bool) string {
	if tailwind {
		return `{% extends 'base.html' %}
{% block title %}Sign Up{% endblock %}
{% block content %}
<section class="flex min-h-[50vh] items-center justify-center px-5 py-8">
    <div class="panel w-full max-w-[420px] p-6">
        <h2>Create an account</h2>
        <p class="mb-6 text-body">Fill in the details below to get started.</p>
        <form method="post" action="{% url 'account_signup' %}" class="space-y-4">
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
        <p class="mt-6 text-center text-body">Already have an account? <a href="{% url 'account_login' %}" class="text-heading no-underline hover:text-accent">Sign in</a></p>
    </div>
</section>
{% endblock %}`
	}
	return `{% extends 'base.html' %}
{% block title %}Sign Up{% endblock %}
{% block content %}
<div class="auth-container">
    <div class="auth-card">
        <h2 class="auth-title">Create an account</h2>
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
            <button type="submit" class="button button-primary auth-btn">Create Account</button>
        </form>
        <p class="auth-footer">Already have an account? <a href="{% url 'account_login' %}">Sign in</a></p>
    </div>
</div>
{% endblock %}`
}

func AllAuthLogout(tailwind bool) string {
	if tailwind {
		return `{% extends 'base.html' %}
{% block title %}Sign Out{% endblock %}
{% block content %}
<section class="flex min-h-[50vh] items-center justify-center px-5 py-8">
    <div class="panel w-full max-w-[420px] p-6 text-center">
        <h2>Sign out</h2>
        <p class="mb-6 text-body">Are you sure you want to sign out?</p>
        <form method="post" action="{% url 'account_logout' %}">
            {% csrf_token %}
            <button type="submit" class="` + authSubmitClass + `">Sign Out</button>
        </form>
        <p class="mt-6 text-body"><a href="{% url 'home' %}" class="text-heading no-underline hover:text-accent">Go back home</a></p>
    </div>
</section>
{% endblock %}`
	}
	return `{% extends 'base.html' %}
{% block title %}Sign Out{% endblock %}
{% block content %}
<div class="auth-container">
    <div class="auth-card auth-card--centered">
        <h2 class="auth-title">Sign out</h2>
        <p class="auth-subtitle">Are you sure you want to sign out?</p>
        <form method="post" action="{% url 'account_logout' %}">
            {% csrf_token %}
            <button type="submit" class="button button-primary auth-btn">Sign Out</button>
        </form>
        <p class="auth-footer"><a href="{% url 'home' %}">Go back home</a></p>
    </div>
</div>
{% endblock %}`
}
