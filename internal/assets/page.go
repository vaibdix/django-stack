package assets

type Page string

const (
	PageBase Page = "base.html"

	PageIndex Page = "index.html"
)

const (
	DRFDocsRoute = "swagger-ui"

	NinjaDocsRoute = "api:openapi-view"

	AppListRoute = "book_list"
)

type PageOptions struct {
	AppName string

	Tailwind bool

	API bool

	APIDocsRoute string

	AppRoute string

	HTMX bool

	AllAuth bool

	GlobalTemplates bool
}

func (o PageOptions) appRoute() string {
	if o.AppRoute == "" {
		return AppListRoute
	}
	return o.AppRoute
}

func (o PageOptions) docsRoute() string {
	if o.APIDocsRoute == "" {
		return DRFDocsRoute
	}
	return o.APIDocsRoute
}
