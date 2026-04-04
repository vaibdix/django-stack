package templates

func tailwindApiDocsTemplateContent() string {
	return `{% extends 'base.html' %}
{% block title %}{{ project_name }} - API Documentation{% endblock %}
{% block content %}
<section class="mx-auto max-w-7xl px-4 py-14 sm:px-6 lg:px-8">
    <div class="max-w-3xl">
        <div class="inline-flex items-center rounded-full border border-white/10 bg-white/[0.04] px-4 py-2 text-xs font-medium uppercase tracking-[0.28em] text-white/60">
            API reference
        </div>
        <h1 class="mt-6 text-4xl font-semibold tracking-[-0.05em] text-white sm:text-5xl">A quieter front door for your Django API.</h1>
        <p class="mt-4 text-base leading-8 text-white/58 sm:text-lg">
            Keep docs, browsable endpoints, and admin workflows connected so the scaffold feels like one product rather than a collection of disconnected tools.
        </p>
    </div>

    <div class="mt-12 grid gap-5 lg:grid-cols-[1.2fr,0.8fr]">
        <div class="rounded-[28px] border border-white/10 bg-white/[0.03] p-7">
            <p class="text-xs uppercase tracking-[0.24em] text-white/35">Quick start</p>
            <div class="mt-5 space-y-4 text-sm leading-7 text-white/58">
                <p>1. Run migrations and seed the example book data.</p>
                <p>2. Visit the browsable API at <span class="text-white">/api/v1/</span>.</p>
                <p>3. Use Django admin to inspect or adjust the same records.</p>
            </div>
        </div>
        <div class="rounded-[28px] border border-white/10 bg-white/[0.03] p-7">
            <p class="text-xs uppercase tracking-[0.24em] text-white/35">Included routes</p>
            <div class="mt-5 space-y-3 text-sm text-white/65">
                <div class="flex items-center justify-between rounded-2xl border border-white/8 px-4 py-3">
                    <span class="text-white/45">GET</span>
                    <span>/api/v1/books/</span>
                </div>
                <div class="flex items-center justify-between rounded-2xl border border-white/8 px-4 py-3">
                    <span class="text-white/45">GET</span>
                    <span>/api/v1/books/recent/</span>
                </div>
                <div class="flex items-center justify-between rounded-2xl border border-white/8 px-4 py-3">
                    <span class="text-white/45">UI</span>
                    <span>/admin/ and /{{ app_name }}/</span>
                </div>
            </div>
        </div>
    </div>
</section>
{% endblock %}`
}
