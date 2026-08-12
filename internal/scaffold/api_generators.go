package scaffold

import "github.com/vaibdix/django-cli/internal/config"

type appURLsData struct {
	AppName string
}

func generateAppURLs(appName string) string {
	return render("app_urls.py.tmpl", appURLsData{AppName: appName})
}

type apiPackageData struct {
	AppName  string
	Pydantic bool
}

func generateBookSerializer(appName string) string {
	return render("api_serializers.py.tmpl", apiPackageData{AppName: appName})
}

func generateAPIPackage(appName string, pydantic bool) map[string]string {
	data := apiPackageData{AppName: appName, Pydantic: pydantic}
	return map[string]string{
		"__init__.py":    render("api_init.py.tmpl", data),
		"serializers.py": generateBookSerializer(appName),
		"views.py":       render("api_views.py.tmpl", data),
		"urls.py":        render("api_urls.py.tmpl", data),
	}
}

type bookFormsData struct {
	Tailwind        bool
	RuntimeGenerics bool
}

func generateBookForms(tailwind, runtimeGenerics bool) string {
	return render("forms.py.tmpl", bookFormsData{Tailwind: tailwind, RuntimeGenerics: runtimeGenerics})
}

type pydanticData struct {
	Pydantic bool
}

func generateBookAPI(pydantic bool) string {
	return render("api_views.py.tmpl", apiPackageData{AppName: "catalog", Pydantic: pydantic})
}

func generateBookSchemas() string { return renderStatic("schemas.py.tmpl") }

func generateHtmxSupport() string { return renderStatic("htmx.py.tmpl") }

type bookModelData struct {
	AppName           string
	HTMLViews         bool
	ConstraintKeyword string
}

func generateBookModel(appName string, htmlViews bool, djangoVersion string) string {
	return render("models.py.tmpl", bookModelData{
		AppName:           appName,
		HTMLViews:         htmlViews,
		ConstraintKeyword: config.DjangoConstraintConditionKeyword(djangoVersion),
	})
}

type bookViewsData struct {
	AppName         string
	HTMX            bool
	Pydantic        bool
	RuntimeGenerics bool
}

func generateBookViews(appName string, htmx, pydantic, runtimeGenerics bool) string {
	return render("views.py.tmpl", bookViewsData{
		AppName:         appName,
		HTMX:            htmx,
		Pydantic:        pydantic,
		RuntimeGenerics: runtimeGenerics,
	})
}

func generateAdminPy(runtimeGenerics bool) string {
	return render("admin_book.py.tmpl", struct{ RuntimeGenerics bool }{runtimeGenerics})
}

type sampleDataData struct {
	AppName  string
	Pydantic bool
}

func generateSampleData(appName string, pydantic bool) string {
	return render("sample_data.py.tmpl", sampleDataData{AppName: appName, Pydantic: pydantic})
}
