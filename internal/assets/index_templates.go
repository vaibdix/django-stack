package assets

func tailwindIndexTemplateContent() string {
	return `{% extends 'base.html' %}
{% block title %}{{ project_name|default:"django-stack" }} - Home{% endblock %}
{% block content %}
<section class="ticks relative border-b border-rule">
    <div class="pointer-events-none absolute inset-0 bg-[radial-gradient(60%_50%_at_50%_0%,rgba(192,132,252,0.12),transparent_70%)]"></div>
    <div class="relative mx-auto max-w-2xl px-6 pb-20 pt-16 text-center">
        <div class="badge border border-rule bg-accent-bg text-accent">Production-ready Django starter</div>
        <h1 class="mt-7 text-4xl font-semibold leading-[1.05] tracking-[-0.045em] text-heading md:text-6xl">
            Ship Django products with a cleaner first impression.
        </h1>
        <p class="mx-auto mt-5 max-w-lg text-base leading-7 text-body">
            django-stack gives you the batteries-included power of Django with a more deliberate visual default:
            <strong class="font-medium text-heading">restrained surfaces, crisp hierarchy, and a calmer way to start shipping.</strong>
        </p>
        <div class="mt-9 flex flex-col justify-center gap-3 sm:flex-row">
            <a href="https://docs.djangoproject.com/" class="btn btn-primary">Read the docs</a>
            <a href="/admin/" class="btn btn-secondary">Launch the admin</a>
        </div>
    </div>
</section>

<section class="px-6 py-14">
    <div class="mb-8">
        <h2 class="text-xl font-semibold tracking-[-0.025em] text-heading">Why Django?</h2>
    </div>
    <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
        <div class="card-hover rounded-2xl border border-rule bg-surface p-6">
            <h3 class="text-base font-semibold tracking-[-0.015em] text-heading">Fast development</h3>
            <p class="mt-2 text-sm leading-7 text-body">A batteries-included framework: the admin, the ORM and the auth system are already here.</p>
        </div>
        <div class="card-hover rounded-2xl border border-rule bg-surface p-6">
            <h3 class="text-base font-semibold tracking-[-0.015em] text-heading">Security first</h3>
            <p class="mt-2 text-sm leading-7 text-body">Argon2 hashing, CSRF protection and secure cookie defaults are configured from the first run.</p>
        </div>
    </div>
</section>
{% endblock %}`
}
