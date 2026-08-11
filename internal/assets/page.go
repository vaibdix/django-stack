package assets

type Page string

const (
	PageBase Page = "base.html"

	PageIndex Page = "index.html"

	PageAPIDocs Page = "api-docs.html"
)

type PageOptions struct {
	AppName string

	Tailwind bool

	API bool

	HTMX bool

	AllAuth bool

	GlobalTemplates bool
}
