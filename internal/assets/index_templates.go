package assets

func tailwindIndexTemplateContent() string {
	return `{% extends 'base.html' %}
{% block title %}{{ project_name|default:"django-stack" }} - Ship calmer Django products{% endblock %}
{% block content %}
<section class="relative overflow-hidden">
    <div class="mx-auto max-w-7xl px-4 pb-16 pt-20 sm:px-6 lg:px-8 lg:pb-24 lg:pt-28">
        <div class="mx-auto max-w-4xl text-center">
            <div class="inline-flex items-center rounded-full border border-white/10 bg-white/[0.04] px-4 py-2 text-xs font-medium uppercase tracking-[0.28em] text-white/60">
                Production-ready Django starter
            </div>
            <h1 class="mt-8 text-5xl font-semibold tracking-[-0.07em] text-white sm:text-6xl lg:text-8xl">
                Ship Django products with a cleaner first impression.
            </h1>
            <p class="mx-auto mt-6 max-w-2xl text-base leading-8 text-white/60 sm:text-lg">
                django-stack gives you the batteries-included power of Django with a more deliberate visual default: monochrome surfaces, crisp hierarchy, and a calmer way to start shipping.
            </p>
            <div class="mt-10 flex flex-col justify-center gap-3 sm:flex-row">
                <a href="https://docs.djangoproject.com/" class="inline-flex items-center justify-center rounded-full bg-white px-6 py-3 text-sm font-semibold text-black transition hover:bg-white/90">Read the docs</a>
                <a href="/admin/" class="inline-flex items-center justify-center rounded-full border border-white/12 px-6 py-3 text-sm font-medium text-white/85 transition hover:border-white/25 hover:text-white">Launch the admin</a>
            </div>
        </div>
    </div>

    <div class="mx-auto grid max-w-7xl gap-5 px-4 pb-20 sm:px-6 lg:grid-cols-3 lg:px-8">
        <div class="rounded-[28px] border border-white/10 bg-white/[0.03] p-6">
            <p class="text-xs uppercase tracking-[0.24em] text-white/35">Default UI</p>
            <h2 class="mt-4 text-2xl font-semibold tracking-[-0.04em] text-white">Inspired by nextjs.org</h2>
            <p class="mt-3 text-sm leading-7 text-white/52">A restrained, editorial homepage and app shell that feel intentional from the first run.</p>
        </div>
        <div class="rounded-[28px] border border-white/10 bg-white/[0.03] p-6">
            <p class="text-xs uppercase tracking-[0.24em] text-white/35">Developer flow</p>
            <h2 class="mt-4 text-2xl font-semibold tracking-[-0.04em] text-white">Admin, API, and app routes wired in</h2>
            <p class="mt-3 text-sm leading-7 text-white/52">Use the generated admin, REST example, and templates together instead of stitching together the basics by hand.</p>
        </div>
        <div class="rounded-[28px] border border-white/10 bg-white/[0.03] p-6">
            <p class="text-xs uppercase tracking-[0.24em] text-white/35">Production habits</p>
            <h2 class="mt-4 text-2xl font-semibold tracking-[-0.04em] text-white">Clean defaults, faster decisions</h2>
            <p class="mt-3 text-sm leading-7 text-white/52">Strong scaffolding helps teams focus on data, behavior, and polish instead of repetitive setup.</p>
        </div>
    </div>
</section>
{% endblock %}`
}
