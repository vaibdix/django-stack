package scaffold

type appURLsData struct {
	AppName string
	HTMX    bool
}

func generateAppURLs(appName string, htmx bool) string {
	return render("app_urls.py.tmpl", appURLsData{AppName: appName, HTMX: htmx})
}

func generateBookSerializer() string { return renderStatic("serializers.py.tmpl") }

func generateBookForms(tailwind bool) string {
	return render("forms.py.tmpl", struct{ Tailwind bool }{tailwind})
}

func generateBookAPI() string { return renderStatic("api.py.tmpl") }

func generateBookModel() string { return renderStatic("models.py.tmpl") }

type bookViewsData struct {
	AppName string
	HTMX    bool
}

func generateBookViews(appName string, htmx bool) string {
	return render("views.py.tmpl", bookViewsData{AppName: appName, HTMX: htmx})
}

func generateAdminPy() string { return renderStatic("admin_book.py.tmpl") }

func generateSampleData(appName string) string {
	return render("sample_data.py.tmpl", struct{ AppName string }{appName})
}
