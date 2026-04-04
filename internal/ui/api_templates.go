package ui

// AppTemplateConfig holds the configuration for generating app HTML templates.
type AppTemplateConfig struct {
	AppName  string
	Tailwind bool
	HTMX     bool
}

// generateBookListTemplate returns the book_list.html template content.
// Produces different markup for Tailwind vs plain CSS, with HTMX search bar
// and inline-delete modifications when HTMX is enabled.
func generateBookListTemplate(cfg AppTemplateConfig) string {
	if cfg.Tailwind {
		return generateBookListTemplateTailwind(cfg)
	}
	return generateBookListTemplatePlain(cfg)
}

func generateBookListTemplateTailwind(cfg AppTemplateConfig) string {
	// HTMX-specific snippets for Tailwind book list
	twSearchBar := ""
	twDeleteBtn := `<a href="/` + cfg.AppName + `/delete/{{ book.id }}/" class="inline-flex flex-1 items-center justify-center rounded-full bg-white/[0.06] px-4 py-2.5 text-sm font-medium text-white transition hover:bg-white/[0.1]">
                        Delete
                    </a>`
	twGridOpen := `<div class="grid gap-5 md:grid-cols-2 xl:grid-cols-3">`
	twArticleOpen := `<article class="group rounded-[28px] border border-white/10 bg-white/[0.03] p-6 shadow-[0_0_0_1px_rgba(255,255,255,0.02)] transition duration-300 hover:-translate-y-0.5 hover:border-white/20 hover:bg-white/[0.05]">`
	if cfg.HTMX {
		twSearchBar = `
        <div class="mb-8">
            <input type="search" name="q" placeholder="Search books by title..."
                   hx-get="/` + cfg.AppName + `/search/"
                   hx-trigger="keyup changed delay:300ms"
                   hx-target="#book-list"
                   hx-swap="innerHTML"
                   class="w-full rounded-2xl border border-white/10 bg-white/[0.03] px-4 py-3 text-sm text-white placeholder:text-white/35 outline-none transition focus:border-white/30 focus:bg-white/[0.05] focus:ring-2 focus:ring-white/10">
        </div>
`
		twDeleteBtn = `<button hx-post="/` + cfg.AppName + `/delete/{{ book.id }}/" hx-target="closest article" hx-swap="outerHTML" hx-confirm="Delete this book?" class="inline-flex flex-1 items-center justify-center rounded-full bg-white/[0.06] px-4 py-2.5 text-sm font-medium text-white transition hover:bg-white/[0.1] border-0 cursor-pointer">
                        Delete
                    </button>`
		twGridOpen = `<div id="book-list" class="grid gap-5 md:grid-cols-2 xl:grid-cols-3">`
		twArticleOpen = `<article id="book-{{ book.id }}" class="group rounded-[28px] border border-white/10 bg-white/[0.03] p-6 shadow-[0_0_0_1px_rgba(255,255,255,0.02)] transition duration-300 hover:-translate-y-0.5 hover:border-white/20 hover:bg-white/[0.05]">`
	}

	return `{% extends 'base.html' %}
{% block title %}` + cfg.AppName + ` - Book Library{% endblock %}
{% block content %}
<section class="relative overflow-hidden">
    <div class="absolute inset-x-0 top-0 h-72 bg-gradient-to-b from-white/10 via-white/5 to-transparent opacity-60"></div>
    <div class="mx-auto max-w-7xl px-4 py-14 sm:px-6 lg:px-8">
        <div class="mb-10 flex flex-col gap-8 lg:flex-row lg:items-end lg:justify-between">
            <div class="max-w-3xl">
                <div class="inline-flex items-center rounded-full border border-white/10 bg-white/[0.04] px-4 py-2 text-xs font-medium uppercase tracking-[0.24em] text-white/65">
                    Example CRUD workspace
                </div>
                <h1 class="mt-6 text-4xl font-semibold tracking-[-0.04em] text-white sm:text-5xl lg:text-6xl">
                    Book inventory with a calmer, sharper control surface.
                </h1>
                <p class="mt-4 max-w-2xl text-base leading-7 text-white/60 sm:text-lg">
                    Manage titles, authors, pricing, and publishing dates from a deliberately minimal library dashboard inspired by the rhythm and restraint of nextjs.org.
                </p>
            </div>
            <div class="grid gap-3 sm:grid-cols-2">
                <div class="rounded-3xl border border-white/10 bg-white/[0.04] p-5 backdrop-blur">
                    <p class="text-xs uppercase tracking-[0.24em] text-white/40">Books tracked</p>
                    <p class="mt-3 text-3xl font-semibold tracking-[-0.04em] text-white">{{ books|length }}</p>
                    <p class="mt-2 text-sm text-white/45">Sample data and manual entries appear here automatically.</p>
                </div>
                <div class="rounded-3xl border border-white/10 bg-white/[0.04] p-5 backdrop-blur">
                    <p class="text-xs uppercase tracking-[0.24em] text-white/40">Admin ready</p>
                    <p class="mt-3 text-3xl font-semibold tracking-[-0.04em] text-white">Books</p>
                    <p class="mt-2 text-sm text-white/45">Use Django admin for audit-friendly edits and search.</p>
                </div>
            </div>
        </div>

        <div class="mb-8 flex flex-col gap-4 border-y border-white/10 py-5 sm:flex-row sm:items-center sm:justify-between">
            <div>
                <p class="text-sm font-medium text-white">Library overview</p>
                <p class="mt-1 text-sm text-white/45">A clean list of every title in your example catalog.</p>
            </div>
            <div class="flex flex-wrap gap-3">
                <a href="/admin/" class="inline-flex items-center justify-center rounded-full border border-white/12 px-5 py-2.5 text-sm font-medium text-white/80 transition hover:border-white/25 hover:text-white">
                    Open admin
                </a>
                <a href="/` + cfg.AppName + `/create/" class="inline-flex items-center justify-center rounded-full bg-white px-5 py-2.5 text-sm font-semibold text-black transition hover:bg-white/90">
                    Add book
                </a>
            </div>
        </div>
` + twSearchBar + `
        {% if messages %}
        <div class="mb-8 space-y-3">
            {% for message in messages %}
            <div class="rounded-2xl border border-white/10 bg-white/[0.04] px-4 py-3 text-sm text-white/75 backdrop-blur">
                {{ message }}
            </div>
            {% endfor %}
        </div>
        {% endif %}

        {% if books %}
        ` + twGridOpen + `
            {% for book in books %}
            ` + twArticleOpen + `
                <div class="flex items-start justify-between gap-4">
                    <div>
                        <p class="text-xs uppercase tracking-[0.24em] text-white/35">Featured title</p>
                        <h2 class="mt-3 text-2xl font-semibold tracking-[-0.04em] text-white">{{ book.title }}</h2>
                    </div>
                    <div class="rounded-full border border-white/10 px-3 py-1 text-sm font-medium text-white/70">
                        ${{ book.price }}
                    </div>
                </div>
                <div class="mt-8 space-y-4 text-sm text-white/52">
                    <div class="flex items-center justify-between gap-4 border-b border-white/6 pb-3">
                        <span>Author</span>
                        <span class="text-right text-white/80">{{ book.author }}</span>
                    </div>
                    <div class="flex items-center justify-between gap-4 border-b border-white/6 pb-3">
                        <span>ISBN</span>
                        <span class="text-right text-white/80">{{ book.isbn }}</span>
                    </div>
                    <div class="flex items-center justify-between gap-4">
                        <span>Published</span>
                        <span class="text-right text-white/80">{{ book.publication_date }}</span>
                    </div>
                </div>
                <div class="mt-8 flex gap-3">
                    <a href="/` + cfg.AppName + `/edit/{{ book.id }}/" class="inline-flex flex-1 items-center justify-center rounded-full border border-white/12 px-4 py-2.5 text-sm font-medium text-white/80 transition hover:border-white/25 hover:text-white">
                        Edit
                    </a>
                    ` + twDeleteBtn + `
                </div>
            </article>
            {% endfor %}
        </div>
        {% else %}
        <div class="rounded-[32px] border border-dashed border-white/12 bg-white/[0.03] px-6 py-14 text-center">
            <div class="mx-auto max-w-xl">
                <p class="text-xs uppercase tracking-[0.3em] text-white/35">Empty state</p>
                <h3 class="mt-4 text-3xl font-semibold tracking-[-0.04em] text-white">No books yet.</h3>
                <p class="mt-4 text-base leading-7 text-white/52">
                    Start with a single title and the library immediately becomes visible in both the custom UI and the Django admin.
                </p>
                <div class="mt-8 flex flex-col justify-center gap-3 sm:flex-row">
                    <a href="/` + cfg.AppName + `/create/" class="inline-flex items-center justify-center rounded-full bg-white px-5 py-3 text-sm font-semibold text-black transition hover:bg-white/90">
                        Add your first book
                    </a>
                    <a href="/admin/" class="inline-flex items-center justify-center rounded-full border border-white/12 px-5 py-3 text-sm font-medium text-white/80 transition hover:border-white/25 hover:text-white">
                        Open Django admin
                    </a>
                </div>
            </div>
        </div>
        {% endif %}
    </div>
</section>
{% endblock %}`
}

func generateBookListTemplatePlain(cfg AppTemplateConfig) string {
	htmxSearchBar := ""
	if cfg.HTMX {
		htmxSearchBar = `
    <div style="margin-bottom:2rem">
        <input type="search" name="q" placeholder="Search books by title..."
               hx-get="/` + cfg.AppName + `/search/"
               hx-trigger="keyup changed delay:300ms"
               hx-target="#book-list"
               hx-swap="innerHTML"
               style="width:100%;border-radius:1rem;border:1px solid rgba(255,255,255,0.1);background:rgba(255,255,255,0.03);padding:0.75rem 1rem;font-size:0.875rem;color:white;outline:none;">
    </div>
`
	}

	divIdAttr := ""
	if cfg.HTMX {
		divIdAttr = ` id="book-list"`
	}

	articleIdAttr := ""
	if cfg.HTMX {
		articleIdAttr = ` id="book-{{ book.id }}"`
	}

	deleteAction := `<a href="/` + cfg.AppName + `/delete/{{ book.id }}/" class="btn-card-delete">Delete</a>`
	if cfg.HTMX {
		deleteAction = `<button hx-post="/` + cfg.AppName + `/delete/{{ book.id }}/" hx-target="closest article" hx-swap="outerHTML" hx-confirm="Delete this book?" class="btn-card-delete" style="cursor:pointer">Delete</button>`
	}

	return `{% extends 'base.html' %}
{% block title %}` + cfg.AppName + ` - Book Library{% endblock %}
{% block extra_head %}
<style>
.book-page { max-width: 1280px; margin: 0 auto; padding: 3.5rem 1rem; }
.book-page-header { display: flex; flex-direction: column; gap: 2rem; margin-bottom: 2.5rem; }
@media (min-width: 1024px) { .book-page-header { flex-direction: row; align-items: flex-end; justify-content: space-between; } }
.book-page-badge { display: inline-flex; align-items: center; border: 1px solid rgba(255,255,255,0.08); background: rgba(255,255,255,0.04); padding: 0.5rem 1rem; border-radius: 9999px; font-size: 0.72rem; letter-spacing: 0.24em; text-transform: uppercase; color: rgba(255,255,255,0.65); }
.book-page-title { margin-top: 1.5rem; font-size: clamp(2rem, 5vw, 3.5rem); font-weight: 600; letter-spacing: -0.04em; line-height: 1.05; color: var(--color-white); }
.book-page-desc { margin-top: 1rem; max-width: 42rem; font-size: 1rem; line-height: 1.75; color: rgba(255,255,255,0.6); }
.book-stats { display: grid; gap: 0.75rem; grid-template-columns: repeat(2, 1fr); }
.book-stat-card { border: 1px solid rgba(255,255,255,0.08); background: rgba(255,255,255,0.04); border-radius: 1.5rem; padding: 1.25rem; }
.book-stat-label { font-size: 0.72rem; text-transform: uppercase; letter-spacing: 0.24em; color: rgba(255,255,255,0.4); }
.book-stat-value { margin-top: 0.75rem; font-size: 1.875rem; font-weight: 600; letter-spacing: -0.04em; color: var(--color-white); }
.book-stat-hint { margin-top: 0.5rem; font-size: 0.875rem; color: rgba(255,255,255,0.45); }
.book-toolbar { display: flex; flex-direction: column; gap: 1rem; border-top: 1px solid rgba(255,255,255,0.08); border-bottom: 1px solid rgba(255,255,255,0.08); padding: 1.25rem 0; margin-bottom: 2rem; }
@media (min-width: 640px) { .book-toolbar { flex-direction: row; align-items: center; justify-content: space-between; } }
.book-toolbar-title { font-size: 0.875rem; font-weight: 500; color: var(--color-white); }
.book-toolbar-sub { margin-top: 0.25rem; font-size: 0.875rem; color: rgba(255,255,255,0.45); }
.book-toolbar-actions { display: flex; flex-wrap: wrap; gap: 0.75rem; }
.btn-primary { display: inline-flex; align-items: center; justify-content: center; padding: 0.625rem 1.25rem; border-radius: 9999px; background: var(--color-white); color: var(--color-black); font-size: 0.875rem; font-weight: 600; text-decoration: none; border: none; cursor: pointer; transition: background 0.2s; }
.btn-primary:hover { background: rgba(255,255,255,0.9); }
.btn-secondary { display: inline-flex; align-items: center; justify-content: center; padding: 0.625rem 1.25rem; border-radius: 9999px; background: transparent; color: rgba(255,255,255,0.8); font-size: 0.875rem; font-weight: 500; text-decoration: none; border: 1px solid rgba(255,255,255,0.12); cursor: pointer; transition: border-color 0.2s, color 0.2s; }
.btn-secondary:hover { border-color: rgba(255,255,255,0.25); color: var(--color-white); }
.book-message { border: 1px solid rgba(255,255,255,0.08); background: rgba(255,255,255,0.04); border-radius: 1rem; padding: 0.75rem 1rem; font-size: 0.875rem; color: rgba(255,255,255,0.75); margin-bottom: 0.75rem; }
.book-grid { display: grid; gap: 1.25rem; }
@media (min-width: 768px) { .book-grid { grid-template-columns: repeat(2, 1fr); } }
@media (min-width: 1280px) { .book-grid { grid-template-columns: repeat(3, 1fr); } }
.book-card { border: 1px solid rgba(255,255,255,0.08); background: rgba(255,255,255,0.03); border-radius: 1.75rem; padding: 1.5rem; transition: border-color 0.3s, background 0.3s, transform 0.3s; }
.book-card:hover { border-color: rgba(255,255,255,0.18); background: rgba(255,255,255,0.05); transform: translateY(-2px); }
.book-card-header { display: flex; align-items: flex-start; justify-content: space-between; gap: 1rem; }
.book-card-label { font-size: 0.72rem; text-transform: uppercase; letter-spacing: 0.24em; color: rgba(255,255,255,0.35); }
.book-card-title { margin-top: 0.75rem; font-size: 1.5rem; font-weight: 600; letter-spacing: -0.04em; color: var(--color-white); }
.book-card-price { border: 1px solid rgba(255,255,255,0.08); border-radius: 9999px; padding: 0.25rem 0.75rem; font-size: 0.875rem; font-weight: 500; color: rgba(255,255,255,0.7); white-space: nowrap; }
.book-card-details { margin-top: 2rem; }
.book-card-row { display: flex; align-items: center; justify-content: space-between; gap: 1rem; padding: 0.75rem 0; font-size: 0.875rem; color: rgba(255,255,255,0.52); }
.book-card-row:not(:last-child) { border-bottom: 1px solid rgba(255,255,255,0.06); }
.book-card-row-value { text-align: right; color: rgba(255,255,255,0.8); }
.book-card-actions { display: flex; gap: 0.75rem; margin-top: 2rem; }
.book-card-actions a { flex: 1; text-align: center; }
.btn-card-edit { display: inline-flex; align-items: center; justify-content: center; padding: 0.625rem 1rem; border-radius: 9999px; border: 1px solid rgba(255,255,255,0.12); background: transparent; color: rgba(255,255,255,0.8); font-size: 0.875rem; font-weight: 500; text-decoration: none; transition: border-color 0.2s, color 0.2s; }
.btn-card-edit:hover { border-color: rgba(255,255,255,0.25); color: var(--color-white); }
.btn-card-delete { display: inline-flex; align-items: center; justify-content: center; padding: 0.625rem 1rem; border-radius: 9999px; border: none; background: rgba(255,255,255,0.06); color: var(--color-white); font-size: 0.875rem; font-weight: 500; text-decoration: none; transition: background 0.2s; }
.btn-card-delete:hover { background: rgba(255,255,255,0.1); }
.book-empty { border: 1px dashed rgba(255,255,255,0.12); background: rgba(255,255,255,0.03); border-radius: 2rem; padding: 3.5rem 1.5rem; text-align: center; }
.book-empty-label { font-size: 0.72rem; text-transform: uppercase; letter-spacing: 0.3em; color: rgba(255,255,255,0.35); }
.book-empty-title { margin-top: 1rem; font-size: 1.875rem; font-weight: 600; letter-spacing: -0.04em; color: var(--color-white); }
.book-empty-desc { margin-top: 1rem; font-size: 1rem; line-height: 1.75; color: rgba(255,255,255,0.52); max-width: 32rem; margin-left: auto; margin-right: auto; }
.book-empty-actions { display: flex; flex-direction: column; gap: 0.75rem; justify-content: center; margin-top: 2rem; }
@media (min-width: 640px) { .book-empty-actions { flex-direction: row; } }
</style>
{% endblock %}
{% block content %}
<section class="book-page">
    <div class="book-page-header">
        <div>
            <div class="book-page-badge">Example CRUD workspace</div>
            <h1 class="book-page-title">Book inventory with a calmer, sharper control surface.</h1>
            <p class="book-page-desc">Manage titles, authors, pricing, and publishing dates from a deliberately minimal library dashboard.</p>
        </div>
        <div class="book-stats">
            <div class="book-stat-card">
                <p class="book-stat-label">Books tracked</p>
                <p class="book-stat-value">{{ books|length }}</p>
                <p class="book-stat-hint">Sample data and manual entries appear here automatically.</p>
            </div>
            <div class="book-stat-card">
                <p class="book-stat-label">Admin ready</p>
                <p class="book-stat-value">Books</p>
                <p class="book-stat-hint">Use Django admin for audit-friendly edits and search.</p>
            </div>
        </div>
    </div>

    <div class="book-toolbar">
        <div>
            <p class="book-toolbar-title">Library overview</p>
            <p class="book-toolbar-sub">A clean list of every title in your example catalog.</p>
        </div>
        <div class="book-toolbar-actions">
            <a href="/admin/" class="btn-secondary">Open admin</a>
            <a href="/` + cfg.AppName + `/create/" class="btn-primary">Add book</a>
        </div>
    </div>
` + htmxSearchBar + `
    {% if messages %}
    <div style="margin-bottom:2rem">
        {% for message in messages %}
        <div class="book-message">{{ message }}</div>
        {% endfor %}
    </div>
    {% endif %}

    {% if books %}
    <div` + divIdAttr + ` class="book-grid">
        {% for book in books %}
        <article` + articleIdAttr + ` class="book-card">
            <div class="book-card-header">
                <div>
                    <p class="book-card-label">Featured title</p>
                    <h2 class="book-card-title">{{ book.title }}</h2>
                </div>
                <div class="book-card-price">${{ book.price }}</div>
            </div>
            <div class="book-card-details">
                <div class="book-card-row"><span>Author</span><span class="book-card-row-value">{{ book.author }}</span></div>
                <div class="book-card-row"><span>ISBN</span><span class="book-card-row-value">{{ book.isbn }}</span></div>
                <div class="book-card-row"><span>Published</span><span class="book-card-row-value">{{ book.publication_date }}</span></div>
            </div>
            <div class="book-card-actions">
                <a href="/` + cfg.AppName + `/edit/{{ book.id }}/" class="btn-card-edit">Edit</a>
                ` + deleteAction + `
            </div>
        </article>
        {% endfor %}
    </div>
    {% else %}
    <div class="book-empty">
        <p class="book-empty-label">Empty state</p>
        <h3 class="book-empty-title">No books yet.</h3>
        <p class="book-empty-desc">Start with a single title and the library immediately becomes visible in both the custom UI and the Django admin.</p>
        <div class="book-empty-actions">
            <a href="/` + cfg.AppName + `/create/" class="btn-primary">Add your first book</a>
            <a href="/admin/" class="btn-secondary">Open Django admin</a>
        </div>
    </div>
    {% endif %}
</section>
{% endblock %}`
}

// generateBookFormTemplate returns the book_form.html template content.
// Produces different markup for Tailwind vs plain CSS.
func generateBookFormTemplate(cfg AppTemplateConfig) string {
	if cfg.Tailwind {
		return `{% extends 'base.html' %}
{% block title %}{{ action }} Book - ` + cfg.AppName + `{% endblock %}
{% block content %}
<section class="mx-auto max-w-6xl px-4 py-14 sm:px-6 lg:px-8">
    <div class="grid gap-8 lg:grid-cols-[0.9fr,1.4fr]">
        <div class="rounded-[28px] border border-white/10 bg-white/[0.03] p-8">
            <p class="text-xs uppercase tracking-[0.28em] text-white/35">Structured editor</p>
            <h1 class="mt-5 text-4xl font-semibold tracking-[-0.05em] text-white">{{ action }} Book</h1>
            <p class="mt-4 text-sm leading-7 text-white/55">
                Keep the catalog clean with consistent title, author, ISBN, publishing date, and pricing data. Everything here is designed to mirror the calmer visual language of the rest of the scaffold.
            </p>
            <div class="mt-8 rounded-3xl border border-white/8 bg-black/40 p-5">
                <p class="text-xs uppercase tracking-[0.24em] text-white/35">Tip</p>
                <p class="mt-3 text-sm leading-6 text-white/60">
                    Once saved, the record is immediately available in both the custom UI and Django admin for faster review.
                </p>
            </div>
        </div>

        <form method="post" class="rounded-[28px] border border-white/10 bg-white/[0.03] p-8 shadow-[0_0_0_1px_rgba(255,255,255,0.02)]">
            {% csrf_token %}
            <div class="space-y-6">
                <div class="space-y-2">
                    <label class="text-sm font-medium text-white/82">Title</label>
                    {{ form.title }}
                </div>
                <div class="space-y-2">
                    <label class="text-sm font-medium text-white/82">Author</label>
                    {{ form.author }}
                </div>
                <div class="space-y-2">
                    <label class="text-sm font-medium text-white/82">ISBN</label>
                    {{ form.isbn }}
                </div>
                <div class="grid gap-6 sm:grid-cols-2">
                    <div class="space-y-2">
                        <label class="text-sm font-medium text-white/82">Publication date</label>
                        {{ form.publication_date }}
                    </div>
                    <div class="space-y-2">
                        <label class="text-sm font-medium text-white/82">Price (USD)</label>
                        {{ form.price }}
                    </div>
                </div>
            </div>
            <div class="mt-8 flex flex-col gap-3 border-t border-white/8 pt-6 sm:flex-row">
                <a href="/` + cfg.AppName + `/" class="inline-flex flex-1 items-center justify-center rounded-full border border-white/12 px-5 py-3 text-sm font-medium text-white/80 transition hover:border-white/25 hover:text-white">
                    Cancel
                </a>
                <button type="submit" class="inline-flex flex-1 items-center justify-center rounded-full bg-white px-5 py-3 text-sm font-semibold text-black transition hover:bg-white/90">
                    {{ action }} Book
                </button>
            </div>
        </form>
    </div>
</section>
{% endblock %}`
	}
	return `{% extends 'base.html' %}
{% block title %}{{ action }} Book - ` + cfg.AppName + `{% endblock %}
{% block extra_head %}
<style>
.form-page { max-width: 72rem; margin: 0 auto; padding: 3.5rem 1rem; }
.form-grid { display: grid; gap: 2rem; }
@media (min-width: 1024px) { .form-grid { grid-template-columns: 0.9fr 1.4fr; } }
.form-sidebar { border: 1px solid rgba(255,255,255,0.08); background: rgba(255,255,255,0.03); border-radius: 1.75rem; padding: 2rem; }
.form-sidebar-label { font-size: 0.72rem; text-transform: uppercase; letter-spacing: 0.28em; color: rgba(255,255,255,0.35); }
.form-sidebar-title { margin-top: 1.25rem; font-size: 2rem; font-weight: 600; letter-spacing: -0.05em; color: var(--color-white); }
.form-sidebar-desc { margin-top: 1rem; font-size: 0.875rem; line-height: 1.75; color: rgba(255,255,255,0.55); }
.form-sidebar-tip { margin-top: 2rem; border: 1px solid rgba(255,255,255,0.06); background: rgba(0,0,0,0.4); border-radius: 1.5rem; padding: 1.25rem; }
.form-sidebar-tip-label { font-size: 0.72rem; text-transform: uppercase; letter-spacing: 0.24em; color: rgba(255,255,255,0.35); }
.form-sidebar-tip-text { margin-top: 0.75rem; font-size: 0.875rem; line-height: 1.5; color: rgba(255,255,255,0.6); }
.form-panel { border: 1px solid rgba(255,255,255,0.08); background: rgba(255,255,255,0.03); border-radius: 1.75rem; padding: 2rem; }
.form-fields { display: flex; flex-direction: column; gap: 1.5rem; }
.form-field label { display: block; font-size: 0.875rem; font-weight: 500; color: rgba(255,255,255,0.82); margin-bottom: 0.5rem; }
.form-panel input, .form-panel select, .form-panel textarea { width: 100%; border-radius: 1rem; border: 1px solid rgba(255,255,255,0.1); background: rgba(255,255,255,0.03); padding: 0.75rem 1rem; font-size: 0.875rem; color: white; outline: none; transition: border-color 0.2s, background 0.2s; }
.form-panel input:focus, .form-panel select:focus, .form-panel textarea:focus { border-color: rgba(255,255,255,0.3); background: rgba(255,255,255,0.05); }
.form-panel input::placeholder { color: rgba(255,255,255,0.35); }
.form-row { display: grid; gap: 1.5rem; }
@media (min-width: 640px) { .form-row { grid-template-columns: 1fr 1fr; } }
.form-actions { display: flex; flex-direction: column; gap: 0.75rem; margin-top: 2rem; padding-top: 1.5rem; border-top: 1px solid rgba(255,255,255,0.06); }
@media (min-width: 640px) { .form-actions { flex-direction: row; } }
.form-actions a, .form-actions button { flex: 1; text-align: center; }
.btn-primary { display: inline-flex; align-items: center; justify-content: center; padding: 0.75rem 1.25rem; border-radius: 9999px; background: var(--color-white); color: var(--color-black); font-size: 0.875rem; font-weight: 600; text-decoration: none; border: none; cursor: pointer; transition: background 0.2s; }
.btn-primary:hover { background: rgba(255,255,255,0.9); }
.btn-secondary { display: inline-flex; align-items: center; justify-content: center; padding: 0.75rem 1.25rem; border-radius: 9999px; background: transparent; color: rgba(255,255,255,0.8); font-size: 0.875rem; font-weight: 500; text-decoration: none; border: 1px solid rgba(255,255,255,0.12); cursor: pointer; transition: border-color 0.2s, color 0.2s; }
.btn-secondary:hover { border-color: rgba(255,255,255,0.25); color: var(--color-white); }
</style>
{% endblock %}
{% block content %}
<section class="form-page">
    <div class="form-grid">
        <div class="form-sidebar">
            <p class="form-sidebar-label">Structured editor</p>
            <h1 class="form-sidebar-title">{{ action }} Book</h1>
            <p class="form-sidebar-desc">Keep the catalog clean with consistent title, author, ISBN, publishing date, and pricing data.</p>
            <div class="form-sidebar-tip">
                <p class="form-sidebar-tip-label">Tip</p>
                <p class="form-sidebar-tip-text">Once saved, the record is immediately available in both the custom UI and Django admin for faster review.</p>
            </div>
        </div>

        <form method="post" class="form-panel">
            {% csrf_token %}
            <div class="form-fields">
                <div class="form-field">
                    <label>Title</label>
                    {{ form.title }}
                </div>
                <div class="form-field">
                    <label>Author</label>
                    {{ form.author }}
                </div>
                <div class="form-field">
                    <label>ISBN</label>
                    {{ form.isbn }}
                </div>
                <div class="form-row">
                    <div class="form-field">
                        <label>Publication date</label>
                        {{ form.publication_date }}
                    </div>
                    <div class="form-field">
                        <label>Price (USD)</label>
                        {{ form.price }}
                    </div>
                </div>
            </div>
            <div class="form-actions">
                <a href="/` + cfg.AppName + `/" class="btn-secondary">Cancel</a>
                <button type="submit" class="btn-primary">{{ action }} Book</button>
            </div>
        </form>
    </div>
</section>
{% endblock %}`
}

// generateBookDeleteTemplate returns the book_delete.html template content.
// Produces different markup for Tailwind vs plain CSS.
func generateBookDeleteTemplate(cfg AppTemplateConfig) string {
	if cfg.Tailwind {
		return `{% extends 'base.html' %}
{% block title %}Delete Book - ` + cfg.AppName + `{% endblock %}
{% block content %}
<section class="mx-auto flex min-h-[70vh] max-w-4xl items-center px-4 py-16 sm:px-6 lg:px-8">
    <div class="w-full rounded-[32px] border border-red-500/20 bg-gradient-to-b from-red-500/10 via-white/[0.03] to-white/[0.02] p-8 text-center sm:p-12">
        <div class="mx-auto flex h-16 w-16 items-center justify-center rounded-full border border-red-400/20 bg-red-500/10">
            <svg xmlns="http://www.w3.org/2000/svg" width="26" height="26" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round" class="text-red-300"><path d="M3 6h18"/><path d="M19 6v14c0 1-1 2-2 2H7c-1 0-2-1-2-2V6"/><path d="M8 6V4c0-1 1-2 2-2h4c1 0 2 1 2 2v2"/><line x1="10" x2="10" y1="11" y2="17"/><line x1="14" x2="14" y1="11" y2="17"/></svg>
        </div>
        <p class="mt-6 text-xs uppercase tracking-[0.3em] text-red-200/60">Confirm removal</p>
        <h2 class="mt-4 text-4xl font-semibold tracking-[-0.05em] text-white">Delete this book?</h2>
        <p class="mx-auto mt-4 max-w-2xl text-sm leading-7 text-white/58 sm:text-base">
            You are about to remove <span class="font-medium text-white">{{ book.title }}</span>. The book disappears from the custom library UI and from Django admin immediately.
        </p>

        <form method="post" class="mx-auto mt-10 flex max-w-xl flex-col gap-3 sm:flex-row">
            {% csrf_token %}
            <a href="/` + cfg.AppName + `/" class="inline-flex flex-1 items-center justify-center rounded-full border border-white/12 px-5 py-3 text-sm font-medium text-white/80 transition hover:border-white/25 hover:text-white">
                Keep book
            </a>
            <button type="submit" class="inline-flex flex-1 items-center justify-center rounded-full bg-red-500 px-5 py-3 text-sm font-semibold text-white transition hover:bg-red-400">
                Delete permanently
            </button>
        </form>
    </div>
</section>
{% endblock %}`
	}
	return `{% extends 'base.html' %}
{% block title %}Delete Book - ` + cfg.AppName + `{% endblock %}
{% block extra_head %}
<style>
.delete-page { display: flex; align-items: center; justify-content: center; min-height: 70vh; max-width: 56rem; margin: 0 auto; padding: 4rem 1rem; }
.delete-card { width: 100%; border: 1px solid rgba(239,68,68,0.2); background: linear-gradient(to bottom, rgba(239,68,68,0.1), rgba(255,255,255,0.03), rgba(255,255,255,0.02)); border-radius: 2rem; padding: 2rem; text-align: center; }
@media (min-width: 640px) { .delete-card { padding: 3rem; } }
.delete-icon { display: flex; align-items: center; justify-content: center; width: 4rem; height: 4rem; margin: 0 auto; border-radius: 9999px; border: 1px solid rgba(248,113,113,0.2); background: rgba(239,68,68,0.1); }
.delete-icon svg { color: #fca5a5; }
.delete-label { margin-top: 1.5rem; font-size: 0.72rem; text-transform: uppercase; letter-spacing: 0.3em; color: rgba(254,202,202,0.6); }
.delete-title { margin-top: 1rem; font-size: 2rem; font-weight: 600; letter-spacing: -0.05em; color: var(--color-white); }
.delete-desc { margin: 1rem auto 0; max-width: 36rem; font-size: 0.875rem; line-height: 1.75; color: rgba(255,255,255,0.58); }
.delete-desc strong { font-weight: 500; color: var(--color-white); }
.delete-actions { display: flex; flex-direction: column; gap: 0.75rem; max-width: 32rem; margin: 2.5rem auto 0; }
@media (min-width: 640px) { .delete-actions { flex-direction: row; } }
.delete-actions a, .delete-actions button { flex: 1; text-align: center; }
.btn-secondary { display: inline-flex; align-items: center; justify-content: center; padding: 0.75rem 1.25rem; border-radius: 9999px; background: transparent; color: rgba(255,255,255,0.8); font-size: 0.875rem; font-weight: 500; text-decoration: none; border: 1px solid rgba(255,255,255,0.12); cursor: pointer; transition: border-color 0.2s, color 0.2s; }
.btn-secondary:hover { border-color: rgba(255,255,255,0.25); color: var(--color-white); }
.btn-danger { display: inline-flex; align-items: center; justify-content: center; padding: 0.75rem 1.25rem; border-radius: 9999px; background: #ef4444; color: white; font-size: 0.875rem; font-weight: 600; border: none; cursor: pointer; transition: background 0.2s; }
.btn-danger:hover { background: #f87171; }
</style>
{% endblock %}
{% block content %}
<section class="delete-page">
    <div class="delete-card">
        <div class="delete-icon">
            <svg xmlns="http://www.w3.org/2000/svg" width="26" height="26" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round"><path d="M3 6h18"/><path d="M19 6v14c0 1-1 2-2 2H7c-1 0-2-1-2-2V6"/><path d="M8 6V4c0-1 1-2 2-2h4c1 0 2 1 2 2v2"/><line x1="10" x2="10" y1="11" y2="17"/><line x1="14" x2="14" y1="11" y2="17"/></svg>
        </div>
        <p class="delete-label">Confirm removal</p>
        <h2 class="delete-title">Delete this book?</h2>
        <p class="delete-desc">You are about to remove <strong>{{ book.title }}</strong>. The book disappears from the custom library UI and from Django admin immediately.</p>

        <form method="post" class="delete-actions">
            {% csrf_token %}
            <a href="/` + cfg.AppName + `/" class="btn-secondary">Keep book</a>
            <button type="submit" class="btn-danger">Delete permanently</button>
        </form>
    </div>
</section>
{% endblock %}`
}

// generateBookListItemsPartial returns the partials/book_list_items.html content.
// This partial is only used when HTMX is enabled.
func generateBookListItemsPartial(cfg AppTemplateConfig) string {
	if cfg.Tailwind {
		return `{% for book in books %}
<article id="book-{{ book.id }}" class="group rounded-[28px] border border-white/10 bg-white/[0.03] p-6 shadow-[0_0_0_1px_rgba(255,255,255,0.02)] transition duration-300 hover:-translate-y-0.5 hover:border-white/20 hover:bg-white/[0.05]">
    <div class="flex items-start justify-between gap-4">
        <div>
            <p class="text-xs uppercase tracking-[0.24em] text-white/35">Featured title</p>
            <h2 class="mt-3 text-2xl font-semibold tracking-[-0.04em] text-white">{{ book.title }}</h2>
        </div>
        <div class="rounded-full border border-white/10 px-3 py-1 text-sm font-medium text-white/70">
            ${{ book.price }}
        </div>
    </div>
    <div class="mt-8 space-y-4 text-sm text-white/52">
        <div class="flex items-center justify-between gap-4 border-b border-white/6 pb-3">
            <span>Author</span>
            <span class="text-right text-white/80">{{ book.author }}</span>
        </div>
        <div class="flex items-center justify-between gap-4 border-b border-white/6 pb-3">
            <span>ISBN</span>
            <span class="text-right text-white/80">{{ book.isbn }}</span>
        </div>
        <div class="flex items-center justify-between gap-4">
            <span>Published</span>
            <span class="text-right text-white/80">{{ book.publication_date }}</span>
        </div>
    </div>
    <div class="mt-8 flex gap-3">
        <a href="/` + cfg.AppName + `/edit/{{ book.id }}/" class="inline-flex flex-1 items-center justify-center rounded-full border border-white/12 px-4 py-2.5 text-sm font-medium text-white/80 transition hover:border-white/25 hover:text-white">
            Edit
        </a>
        <button hx-post="/` + cfg.AppName + `/delete/{{ book.id }}/" hx-target="closest article" hx-swap="outerHTML" hx-confirm="Delete this book?" class="inline-flex flex-1 items-center justify-center rounded-full bg-white/[0.06] px-4 py-2.5 text-sm font-medium text-white transition hover:bg-white/[0.1] border-0 cursor-pointer">
            Delete
        </button>
    </div>
</article>
{% empty %}
<div class="rounded-[32px] border border-dashed border-white/12 bg-white/[0.03] px-6 py-14 text-center col-span-full">
    <p class="text-white/52">No books match your search.</p>
</div>
{% endfor %}
`
	}
	return `{% for book in books %}
<article id="book-{{ book.id }}" class="book-card">
    <div class="book-card-header">
        <div>
            <p class="book-card-label">Featured title</p>
            <h2 class="book-card-title">{{ book.title }}</h2>
        </div>
        <div class="book-card-price">${{ book.price }}</div>
    </div>
    <div class="book-card-details">
        <div class="book-card-row"><span>Author</span><span class="book-card-row-value">{{ book.author }}</span></div>
        <div class="book-card-row"><span>ISBN</span><span class="book-card-row-value">{{ book.isbn }}</span></div>
        <div class="book-card-row"><span>Published</span><span class="book-card-row-value">{{ book.publication_date }}</span></div>
    </div>
    <div class="book-card-actions">
        <a href="/` + cfg.AppName + `/edit/{{ book.id }}/" class="btn-card-edit">Edit</a>
        <button hx-post="/` + cfg.AppName + `/delete/{{ book.id }}/" hx-target="closest article" hx-swap="outerHTML" hx-confirm="Delete this book?" class="btn-card-delete" style="cursor:pointer">Delete</button>
    </div>
</article>
{% empty %}
<div class="book-empty" style="grid-column:1/-1">
    <p style="color:rgba(255,255,255,0.52)">No books match your search.</p>
</div>
{% endfor %}
`
}

// generateBookFormPartial returns the partials/book_form_partial.html content.
// This partial is only used when HTMX is enabled.
func generateBookFormPartial(cfg AppTemplateConfig) string {
	if cfg.Tailwind {
		return `<form method="post" hx-post hx-target="#book-list" hx-swap="innerHTML" class="rounded-[28px] border border-white/10 bg-white/[0.03] p-8 shadow-[0_0_0_1px_rgba(255,255,255,0.02)]">
    {% csrf_token %}
    <div class="space-y-6">
        <div class="space-y-2">
            <label class="text-sm font-medium text-white/82">Title</label>
            {{ form.title }}
        </div>
        <div class="space-y-2">
            <label class="text-sm font-medium text-white/82">Author</label>
            {{ form.author }}
        </div>
        <div class="space-y-2">
            <label class="text-sm font-medium text-white/82">ISBN</label>
            {{ form.isbn }}
        </div>
        <div class="grid gap-6 sm:grid-cols-2">
            <div class="space-y-2">
                <label class="text-sm font-medium text-white/82">Publication date</label>
                {{ form.publication_date }}
            </div>
            <div class="space-y-2">
                <label class="text-sm font-medium text-white/82">Price (USD)</label>
                {{ form.price }}
            </div>
        </div>
    </div>
    <div class="mt-8 flex flex-col gap-3 border-t border-white/8 pt-6 sm:flex-row">
        <a href="/` + cfg.AppName + `/" class="inline-flex flex-1 items-center justify-center rounded-full border border-white/12 px-5 py-3 text-sm font-medium text-white/80 transition hover:border-white/25 hover:text-white">
            Cancel
        </a>
        <button type="submit" class="inline-flex flex-1 items-center justify-center rounded-full bg-white px-5 py-3 text-sm font-semibold text-black transition hover:bg-white/90">
            {{ action }} Book
        </button>
    </div>
</form>
`
	}
	return `<form method="post" hx-post hx-target="#book-list" hx-swap="innerHTML" class="form-panel">
    {% csrf_token %}
    <div class="form-fields">
        <div class="form-field">
            <label>Title</label>
            {{ form.title }}
        </div>
        <div class="form-field">
            <label>Author</label>
            {{ form.author }}
        </div>
        <div class="form-field">
            <label>ISBN</label>
            {{ form.isbn }}
        </div>
        <div class="form-row">
            <div class="form-field">
                <label>Publication date</label>
                {{ form.publication_date }}
            </div>
            <div class="form-field">
                <label>Price (USD)</label>
                {{ form.price }}
            </div>
        </div>
    </div>
    <div class="form-actions">
        <a href="/` + cfg.AppName + `/" class="btn-secondary">Cancel</a>
        <button type="submit" class="btn-primary">{{ action }} Book</button>
    </div>
</form>
`
}
