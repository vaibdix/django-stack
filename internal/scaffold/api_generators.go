package scaffold

import "github.com/vaibdix/django-stack/internal/config"

type appURLsData struct {
	AppName string
}

func generateAppURLs(appName string) string {
	return render("app_urls.py.tmpl", appURLsData{AppName: appName})
}

type apiPackageData struct {
	AppName string
}

func generateAPIPackage(appName string) map[string]string {
	data := apiPackageData{AppName: appName}
	return map[string]string{
		"__init__.py":    render("api_init.py.tmpl", data),
		"serializers.py": render("api_serializers.py.tmpl", data),
		"views.py":       render("api_views.py.tmpl", data),
		"urls.py":        render("api_urls.py.tmpl", data),
	}
}

type ninjaPackageData struct {
	AppName     string
	ProjectName string
}

func generateNinjaPackage(appName, projectName string) map[string]string {
	data := ninjaPackageData{AppName: appName, ProjectName: projectName}
	return map[string]string{
		"__init__.py": render("ninja_init.py.tmpl", data),
		"api.py":      render("ninja_api.py.tmpl", data),
		"router.py":   render("ninja_router.py.tmpl", data),
		"schemas.py":  render("ninja_schemas.py.tmpl", data),
	}
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
	Postgres  bool
}

func generateBookModel(appName string, htmlViews bool, database config.Database) string {
	return render("models.py.tmpl", bookModelData{
		AppName:   appName,
		HTMLViews: htmlViews,
		Postgres:  database.UsesPostgres(),
	})
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
