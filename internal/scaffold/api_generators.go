package scaffold

import "github.com/vaibdix/django-cli/internal/config"

type appURLsData struct {
	AppName string
}

func generateAppURLs(appName string) string {
	return render("app_urls.py.tmpl", appURLsData{AppName: appName})
}

type apiPackageData struct {
	AppName string
}

func generateBookSerializer(appName string) string {
	return render("api_serializers.py.tmpl", apiPackageData{AppName: appName})
}

func generateAPIPackage(appName string) map[string]string {
	data := apiPackageData{AppName: appName}
	return map[string]string{
		"__init__.py":    render("api_init.py.tmpl", data),
		"serializers.py": generateBookSerializer(appName),
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

func generateBookForms(tailwind, runtimeGenerics bool) string {
	return render("forms.py.tmpl", bookFormsData{Tailwind: tailwind, RuntimeGenerics: runtimeGenerics})
}

func generateBookFields(runtimeGenerics bool) string {
	return render("fields.py.tmpl", struct{ RuntimeGenerics bool }{runtimeGenerics})
}

func generateHtmxSupport() string { return renderStatic("htmx.py.tmpl") }

type bookModelData struct {
	AppName           string
	HTMLViews         bool
	Postgres          bool
	ConstraintKeyword string
}

func generateBookModel(appName string, htmlViews bool, djangoVersion string, database config.Database) string {
	return render("models.py.tmpl", bookModelData{
		AppName:           appName,
		HTMLViews:         htmlViews,
		Postgres:          database.UsesPostgres(),
		ConstraintKeyword: config.DjangoConstraintConditionKeyword(djangoVersion),
	})
}

type bookViewsData struct {
	AppName         string
	HTMX            bool
	RuntimeGenerics bool
	Stubs52         bool
}

func generateBookViews(appName string, htmx, runtimeGenerics bool) string {
	return render("views.py.tmpl", bookViewsData{
		AppName:         appName,
		HTMX:            htmx,
		RuntimeGenerics: runtimeGenerics,
	})
}

func generateAdminPy(runtimeGenerics bool) string {
	return render("admin_book.py.tmpl", struct{ RuntimeGenerics bool }{runtimeGenerics})
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
