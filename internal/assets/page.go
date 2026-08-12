package assets

type Page string

const (
	PageBase Page = "base.html"

	PageIndex Page = "index.html"

	PageAPIDocs Page = "api-docs.html"
)

const (
	DRFDocsRoute = "swagger-ui"

	NinjaDocsRoute = "api:openapi-view"
)

type PageOptions struct {
	AppName string

	Tailwind bool

	API bool

	APIDocsRoute string

	HTMX bool

	AllAuth bool

	GlobalTemplates bool
}

func (o PageOptions) docsRoute() string {
	if o.APIDocsRoute == "" {
		return DRFDocsRoute
	}
	return o.APIDocsRoute
}
