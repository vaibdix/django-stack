package assets

// AllAuthLogin returns the django-allauth login page, styled for Tailwind or
// plain CSS.
func AllAuthLogin(tailwind bool) string {
	if tailwind {
		return `{% extends 'base.html' %}
{% block title %}Sign In{% endblock %}
{% block content %}
<section class="flex min-h-[80vh] items-center justify-center px-4">
    <div class="w-full max-w-md rounded-[28px] border border-white/10 bg-white/[0.03] p-8">
        <p class="text-xs uppercase tracking-[0.24em] text-white/35">Account</p>
        <h1 class="mt-4 text-3xl font-semibold tracking-[-0.04em] text-white">Sign in</h1>
        <p class="mt-2 text-sm text-white/52">Welcome back. Enter your credentials to continue.</p>
        <form method="post" action="{% url 'account_login' %}" class="mt-8 space-y-5">
            {% csrf_token %}
            {% if form.non_field_errors %}
            <div class="rounded-2xl border border-red-500/20 bg-red-500/10 px-4 py-3 text-sm text-red-300">
                {% for error in form.non_field_errors %}{{ error }}{% endfor %}
            </div>
            {% endif %}
            <div>
                <label for="id_login" class="block text-xs uppercase tracking-[0.24em] text-white/35">Username or Email</label>
                <input type="text" name="login" id="id_login" value="{{ form.login.value|default:'' }}" class="mt-2 w-full rounded-2xl border border-white/10 bg-white/[0.04] px-4 py-3 text-sm text-white placeholder-white/25 outline-none transition focus:border-white/25" placeholder="you@example.com" required>
                {% if form.login.errors %}<p class="mt-1 text-xs text-red-400">{{ form.login.errors.0 }}</p>{% endif %}
            </div>
            <div>
                <label for="id_password" class="block text-xs uppercase tracking-[0.24em] text-white/35">Password</label>
                <input type="password" name="password" id="id_password" class="mt-2 w-full rounded-2xl border border-white/10 bg-white/[0.04] px-4 py-3 text-sm text-white placeholder-white/25 outline-none transition focus:border-white/25" placeholder="••••••••" required>
                {% if form.password.errors %}<p class="mt-1 text-xs text-red-400">{{ form.password.errors.0 }}</p>{% endif %}
            </div>
            {% if redirect_field_value %}<input type="hidden" name="{{ redirect_field_name }}" value="{{ redirect_field_value }}">{% endif %}
            <button type="submit" class="w-full rounded-full bg-white px-6 py-3 text-sm font-semibold text-black transition hover:bg-white/90">Sign In</button>
        </form>
        <p class="mt-6 text-center text-sm text-white/45">Don't have an account? <a href="{% url 'account_signup' %}" class="text-white/70 transition hover:text-white">Sign up</a></p>
    </div>
</section>
{% endblock %}`
	}
	return `{% extends 'base.html' %}
{% block title %}Sign In{% endblock %}
{% block extra_head %}
<style>
    .auth-container { display: flex; min-height: 80vh; align-items: center; justify-content: center; padding: 1rem; }
    .auth-card { width: 100%; max-width: 28rem; border: 1px solid rgba(255,255,255,0.1); background: rgba(255,255,255,0.03); border-radius: 28px; padding: 2rem; }
    .auth-label { display: block; font-size: 0.7rem; text-transform: uppercase; letter-spacing: 0.24em; color: rgba(255,255,255,0.35); }
    .auth-title { margin-top: 1rem; font-size: 1.875rem; font-weight: 600; letter-spacing: -0.04em; color: #fff; }
    .auth-subtitle { margin-top: 0.5rem; font-size: 0.875rem; color: rgba(255,255,255,0.52); }
    .auth-form { margin-top: 2rem; }
    .auth-form > * + * { margin-top: 1.25rem; }
    .auth-input { margin-top: 0.5rem; width: 100%; border: 1px solid rgba(255,255,255,0.1); background: rgba(255,255,255,0.04); border-radius: 16px; padding: 0.75rem 1rem; font-size: 0.875rem; color: #fff; outline: none; }
    .auth-input:focus { border-color: rgba(255,255,255,0.25); }
    .auth-input::placeholder { color: rgba(255,255,255,0.25); }
    .auth-btn { margin-top: 1.25rem; width: 100%; background: #fff; color: #000; border: none; border-radius: 9999px; padding: 0.75rem 1.5rem; font-size: 0.875rem; font-weight: 600; cursor: pointer; }
    .auth-btn:hover { background: rgba(255,255,255,0.9); }
    .auth-footer { margin-top: 1.5rem; text-align: center; font-size: 0.875rem; color: rgba(255,255,255,0.45); }
    .auth-footer a { color: rgba(255,255,255,0.7); text-decoration: none; }
    .auth-footer a:hover { color: #fff; }
    .auth-error { border: 1px solid rgba(239,68,68,0.2); background: rgba(239,68,68,0.1); border-radius: 16px; padding: 0.75rem 1rem; font-size: 0.875rem; color: #fca5a5; }
    .field-error { margin-top: 0.25rem; font-size: 0.75rem; color: #f87171; }
</style>
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
<section class="flex min-h-[80vh] items-center justify-center px-4">
    <div class="w-full max-w-md rounded-[28px] border border-white/10 bg-white/[0.03] p-8">
        <p class="text-xs uppercase tracking-[0.24em] text-white/35">Get started</p>
        <h1 class="mt-4 text-3xl font-semibold tracking-[-0.04em] text-white">Create an account</h1>
        <p class="mt-2 text-sm text-white/52">Fill in the details below to get started.</p>
        <form method="post" action="{% url 'account_signup' %}" class="mt-8 space-y-5">
            {% csrf_token %}
            {% if form.non_field_errors %}
            <div class="rounded-2xl border border-red-500/20 bg-red-500/10 px-4 py-3 text-sm text-red-300">
                {% for error in form.non_field_errors %}{{ error }}{% endfor %}
            </div>
            {% endif %}
            <div>
                <label for="id_username" class="block text-xs uppercase tracking-[0.24em] text-white/35">Username</label>
                <input type="text" name="username" id="id_username" value="{{ form.username.value|default:'' }}" class="mt-2 w-full rounded-2xl border border-white/10 bg-white/[0.04] px-4 py-3 text-sm text-white placeholder-white/25 outline-none transition focus:border-white/25" placeholder="johndoe" required>
                {% if form.username.errors %}<p class="mt-1 text-xs text-red-400">{{ form.username.errors.0 }}</p>{% endif %}
            </div>
            <div>
                <label for="id_email" class="block text-xs uppercase tracking-[0.24em] text-white/35">Email</label>
                <input type="email" name="email" id="id_email" value="{{ form.email.value|default:'' }}" class="mt-2 w-full rounded-2xl border border-white/10 bg-white/[0.04] px-4 py-3 text-sm text-white placeholder-white/25 outline-none transition focus:border-white/25" placeholder="you@example.com" required>
                {% if form.email.errors %}<p class="mt-1 text-xs text-red-400">{{ form.email.errors.0 }}</p>{% endif %}
            </div>
            <div>
                <label for="id_password1" class="block text-xs uppercase tracking-[0.24em] text-white/35">Password</label>
                <input type="password" name="password1" id="id_password1" class="mt-2 w-full rounded-2xl border border-white/10 bg-white/[0.04] px-4 py-3 text-sm text-white placeholder-white/25 outline-none transition focus:border-white/25" placeholder="••••••••" required>
                {% if form.password1.errors %}<p class="mt-1 text-xs text-red-400">{{ form.password1.errors.0 }}</p>{% endif %}
            </div>
            <div>
                <label for="id_password2" class="block text-xs uppercase tracking-[0.24em] text-white/35">Confirm Password</label>
                <input type="password" name="password2" id="id_password2" class="mt-2 w-full rounded-2xl border border-white/10 bg-white/[0.04] px-4 py-3 text-sm text-white placeholder-white/25 outline-none transition focus:border-white/25" placeholder="••••••••" required>
                {% if form.password2.errors %}<p class="mt-1 text-xs text-red-400">{{ form.password2.errors.0 }}</p>{% endif %}
            </div>
            {% if redirect_field_value %}<input type="hidden" name="{{ redirect_field_name }}" value="{{ redirect_field_value }}">{% endif %}
            <button type="submit" class="w-full rounded-full bg-white px-6 py-3 text-sm font-semibold text-black transition hover:bg-white/90">Create Account</button>
        </form>
        <p class="mt-6 text-center text-sm text-white/45">Already have an account? <a href="{% url 'account_login' %}" class="text-white/70 transition hover:text-white">Sign in</a></p>
    </div>
</section>
{% endblock %}`
	}
	return `{% extends 'base.html' %}
{% block title %}Sign Up{% endblock %}
{% block extra_head %}
<style>
    .auth-container { display: flex; min-height: 80vh; align-items: center; justify-content: center; padding: 1rem; }
    .auth-card { width: 100%; max-width: 28rem; border: 1px solid rgba(255,255,255,0.1); background: rgba(255,255,255,0.03); border-radius: 28px; padding: 2rem; }
    .auth-label { display: block; font-size: 0.7rem; text-transform: uppercase; letter-spacing: 0.24em; color: rgba(255,255,255,0.35); }
    .auth-title { margin-top: 1rem; font-size: 1.875rem; font-weight: 600; letter-spacing: -0.04em; color: #fff; }
    .auth-subtitle { margin-top: 0.5rem; font-size: 0.875rem; color: rgba(255,255,255,0.52); }
    .auth-form { margin-top: 2rem; }
    .auth-form > * + * { margin-top: 1.25rem; }
    .auth-input { margin-top: 0.5rem; width: 100%; border: 1px solid rgba(255,255,255,0.1); background: rgba(255,255,255,0.04); border-radius: 16px; padding: 0.75rem 1rem; font-size: 0.875rem; color: #fff; outline: none; }
    .auth-input:focus { border-color: rgba(255,255,255,0.25); }
    .auth-input::placeholder { color: rgba(255,255,255,0.25); }
    .auth-btn { margin-top: 1.25rem; width: 100%; background: #fff; color: #000; border: none; border-radius: 9999px; padding: 0.75rem 1.5rem; font-size: 0.875rem; font-weight: 600; cursor: pointer; }
    .auth-btn:hover { background: rgba(255,255,255,0.9); }
    .auth-footer { margin-top: 1.5rem; text-align: center; font-size: 0.875rem; color: rgba(255,255,255,0.45); }
    .auth-footer a { color: rgba(255,255,255,0.7); text-decoration: none; }
    .auth-footer a:hover { color: #fff; }
    .auth-error { border: 1px solid rgba(239,68,68,0.2); background: rgba(239,68,68,0.1); border-radius: 16px; padding: 0.75rem 1rem; font-size: 0.875rem; color: #fca5a5; }
    .field-error { margin-top: 0.25rem; font-size: 0.75rem; color: #f87171; }
</style>
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
<section class="flex min-h-[80vh] items-center justify-center px-4">
    <div class="w-full max-w-md rounded-[28px] border border-white/10 bg-white/[0.03] p-8 text-center">
        <p class="text-xs uppercase tracking-[0.24em] text-white/35">Account</p>
        <h1 class="mt-4 text-3xl font-semibold tracking-[-0.04em] text-white">Sign out</h1>
        <p class="mt-2 text-sm text-white/52">Are you sure you want to sign out?</p>
        <form method="post" action="{% url 'account_logout' %}" class="mt-8">
            {% csrf_token %}
            <button type="submit" class="w-full rounded-full bg-white px-6 py-3 text-sm font-semibold text-black transition hover:bg-white/90">Sign Out</button>
        </form>
        <p class="mt-6 text-sm text-white/45"><a href="/" class="text-white/70 transition hover:text-white">Go back home</a></p>
    </div>
</section>
{% endblock %}`
	}
	return `{% extends 'base.html' %}
{% block title %}Sign Out{% endblock %}
{% block extra_head %}
<style>
    .auth-container { display: flex; min-height: 80vh; align-items: center; justify-content: center; padding: 1rem; }
    .auth-card { width: 100%; max-width: 28rem; border: 1px solid rgba(255,255,255,0.1); background: rgba(255,255,255,0.03); border-radius: 28px; padding: 2rem; text-align: center; }
    .auth-label { font-size: 0.7rem; text-transform: uppercase; letter-spacing: 0.24em; color: rgba(255,255,255,0.35); }
    .auth-title { margin-top: 1rem; font-size: 1.875rem; font-weight: 600; letter-spacing: -0.04em; color: #fff; }
    .auth-subtitle { margin-top: 0.5rem; font-size: 0.875rem; color: rgba(255,255,255,0.52); }
    .auth-btn { margin-top: 2rem; width: 100%; background: #fff; color: #000; border: none; border-radius: 9999px; padding: 0.75rem 1.5rem; font-size: 0.875rem; font-weight: 600; cursor: pointer; }
    .auth-btn:hover { background: rgba(255,255,255,0.9); }
    .auth-footer { margin-top: 1.5rem; font-size: 0.875rem; color: rgba(255,255,255,0.45); }
    .auth-footer a { color: rgba(255,255,255,0.7); text-decoration: none; }
    .auth-footer a:hover { color: #fff; }
</style>
{% endblock %}
{% block content %}
<div class="auth-container">
    <div class="auth-card">
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
