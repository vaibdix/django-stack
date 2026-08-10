package assets

// Page identifies which built-in HTML page to render.
//
// The values match the filenames the scaffolder writes, which is what the old
// stringly-typed dispatch keyed on. Making them a named type means a typo is a
// compile error instead of an empty string at runtime.
type Page string

const (
	// PageBase is the shared layout every other template extends.
	PageBase Page = "base.html"
	// PageIndex is the generated landing page.
	PageIndex Page = "index.html"
	// PageAPIDocs is the API documentation shell shown when REST is enabled.
	PageAPIDocs Page = "api-docs.html"
)

// PageOptions describes the feature set a page is rendered for.
//
// It replaces the previous map[string]string, whose booleans were passed as the
// literals "true"/"false" through a boolToString helper. Every field below was
// a map key that no compiler ever checked: a misspelled "tailwnd" silently
// rendered the plain variant.
type PageOptions struct {
	// AppName is the Django app to link to in the nav, if any.
	AppName string
	// Tailwind selects the Tailwind-styled variants.
	Tailwind bool
	// API adds the API documentation link.
	API bool
	// HTMX loads the django-htmx tag library, script tag and CSRF header.
	HTMX bool
	// AllAuth adds the login/signup/logout nav links.
	AllAuth bool
	// GlobalTemplates renders stylesheets as {% static %} links rather than
	// inlining them, which is only correct when a static/ directory exists.
	GlobalTemplates bool
}
