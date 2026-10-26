package scaffold

import "github.com/vaibdix/django-stack/internal/config"

type appTemplateConfig struct {
	AppName  string
	Tailwind bool
	HTMX     bool
}

func (c appTemplateConfig) style() string {
	if c.Tailwind {
		return "tailwind"
	}
	return "plain"
}

func (c appTemplateConfig) templateFor(page string) string {
	return page + "_" + c.style() + ".html.tmpl"
}

func (c appTemplateConfig) render(page string) string {
	return render(c.templateFor(page), c)
}

type appURLsData struct {
	AppName string
}

func generateAppURLs(appName string) string {
	return render("app_urls.py.tmpl", appURLsData{AppName: appName})
}

type bookFormsData struct {
	Tailwind        bool
	RuntimeGenerics bool
}

type typingData struct {
	RuntimeGenerics bool
}

func generateBookForms(data bookFormsData) string {
	return render("forms.py.tmpl", data)
}

func generateBookFields(data typingData) string {
	return render("fields.py.tmpl", data)
}

func generateHtmxSupport() string { return renderStatic("htmx.py.tmpl") }

type bookModelData struct {
	AppName   string
	HTMLViews bool
}

func generateBookModel(appName string, htmlViews bool) string {
	return render("models.py.tmpl", bookModelData{AppName: appName, HTMLViews: htmlViews})
}

type bookViewsData struct {
	AppName         string
	HTMX            bool
	RuntimeGenerics bool
	Stubs52         bool
}

func renderBookViews(data bookViewsData) string {
	return render("views.py.tmpl", data)
}

func generateAdminPy(data typingData) string {
	return render("admin_book.py.tmpl", data)
}

type sampleDataData struct {
	AppName      string
	UpsertTarget bool
}

func generateSampleData(appName string, database config.Database) string {
	return render("sample_data.py.tmpl", sampleDataData{
		AppName:      appName,
		UpsertTarget: database.SupportsUpsertTarget(),
	})
}
