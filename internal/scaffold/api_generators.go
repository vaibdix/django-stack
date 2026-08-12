package scaffold

import "github.com/vaibdix/django-cli/internal/config"

type appURLsData struct {
	AppName string
	HTMX    bool
}

func generateAppURLs(appName string, htmx bool) string {
	return render("app_urls.py.tmpl", appURLsData{AppName: appName, HTMX: htmx})
}

func generateBookSerializer() string { return renderStatic("serializers.py.tmpl") }

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
	return render("api.py.tmpl", pydanticData{Pydantic: pydantic})
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
	AppName  string
	HTMX     bool
	Pydantic bool
}

func generateBookViews(appName string, htmx, pydantic bool) string {
	return render("views.py.tmpl", bookViewsData{AppName: appName, HTMX: htmx, Pydantic: pydantic})
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
