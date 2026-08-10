package scaffold

// Django templates for the example "Book" app.
//
// The markup lives in templates/html/*.html.tmpl. It used to be 684 lines of
// HTML spliced together from Go string literals and a handful of snippet
// variables (twSearchBar, twDeleteBtn, twGridOpen, twArticleOpen) that were
// reassigned in an if-block above the return — so reading the Tailwind list
// page meant tracking four variables through two branches.
//
// Each page exists in a plain and a Tailwind variant, selected by name; HTMX
// differences are expressed inline with [[if .HTMX]] where they exist.

// AppTemplateConfig holds the configuration for generating app HTML templates.
type AppTemplateConfig struct {
	AppName  string
	Tailwind bool
	HTMX     bool
}

// style is the template-name suffix for the configured CSS approach. The two
// variants are separate files rather than one branching template because they
// share almost no markup.
func (c AppTemplateConfig) style() string {
	if c.Tailwind {
		return "tailwind"
	}
	return "plain"
}

// templateFor names the embedded template for a page in the configured style.
func (c AppTemplateConfig) templateFor(page string) string {
	return page + "_" + c.style() + ".html.tmpl"
}

// generateBookListTemplate returns the book_list.html template content.
//
// This is the only page HTMX changes: it gains a live-search input, stable ids
// for the list container and each article, and a delete button that posts and
// swaps in place rather than navigating.
func generateBookListTemplate(cfg AppTemplateConfig) string {
	return render(cfg.templateFor("book_list"), cfg)
}

// generateBookFormTemplate returns the book_form.html template content.
//
// It does not vary with HTMX: under HTMX the views render
// partials/book_form_partial.html instead, so this full-page form is only ever
// served to a non-HTMX request.
func generateBookFormTemplate(cfg AppTemplateConfig) string {
	return render(cfg.templateFor("book_form"), cfg)
}

// generateBookDeleteTemplate returns the book_delete.html confirmation page.
// Like the form page, it is only reached without HTMX.
func generateBookDeleteTemplate(cfg AppTemplateConfig) string {
	return render(cfg.templateFor("book_delete"), cfg)
}

// generateBookListItemsPartial returns the HTMX partial that renders just the
// list rows, used as the swap target for search and delete.
func generateBookListItemsPartial(cfg AppTemplateConfig) string {
	return render(cfg.templateFor("book_list_items"), cfg)
}

// generateBookFormPartial returns the HTMX partial for the create/edit form.
func generateBookFormPartial(cfg AppTemplateConfig) string {
	return render(cfg.templateFor("book_form_partial"), cfg)
}
