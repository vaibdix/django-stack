package scaffold

// Python source for the example "Book" app that ships with the REST Framework
// option. The content lives in templates/python/*.py.tmpl; these functions are
// just the typed entry points to it.
//
// Four of these used to take an appName parameter they never read
// (serializers, api, models, admin). Moving the bodies into templates made that
// obvious — a template with no fields needs no data — so the dead parameters
// are gone rather than being carried forward.

// appURLsData drives the app's urls.py.
type appURLsData struct {
	AppName string
	HTMX    bool
}

// generateAppURLs returns the content of the app's urls.py file.
// When htmx is true, an additional search URL pattern is included.
func generateAppURLs(appName string, htmx bool) string {
	return render("app_urls.py.tmpl", appURLsData{AppName: appName, HTMX: htmx})
}

// generateBookSerializer returns the content of serializers.py.
func generateBookSerializer() string { return renderStatic("serializers.py.tmpl") }

// generateBookForms returns the content of forms.py.
//
// When tailwind is true the widgets share a module-level INPUT_CLASS constant.
// The class list used to be interpolated into each widget, which produced
// 300-character lines; ruff rejects those with E501 and cannot autofix them, so
// the mandatory formatting pass failed and aborted the whole scaffold.
func generateBookForms(tailwind bool) string {
	return render("forms.py.tmpl", struct{ Tailwind bool }{tailwind})
}

// generateBookAPI returns the content of api.py with router configuration.
func generateBookAPI() string { return renderStatic("api.py.tmpl") }

// generateBookModel returns the content of models.py with the Book model.
func generateBookModel() string { return renderStatic("models.py.tmpl") }

// bookViewsData drives the app's views.py.
type bookViewsData struct {
	AppName string
	HTMX    bool
}

// generateBookViews returns the content of views.py.
// When htmx is true, HTMX-aware view logic is included (partials, search, and
// an empty HttpResponse for out-of-band deletes).
func generateBookViews(appName string, htmx bool) string {
	return render("views.py.tmpl", bookViewsData{AppName: appName, HTMX: htmx})
}

// generateAdminPy returns an admin.py registering the Book model.
func generateAdminPy() string { return renderStatic("admin_book.py.tmpl") }

// generateSampleData returns the content of the create_sample_data management
// command.
func generateSampleData(appName string) string {
	return render("sample_data.py.tmpl", struct{ AppName string }{appName})
}
