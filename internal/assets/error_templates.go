package assets

type ErrorPage struct {
	Code    string
	Title   string
	Message string

	Standalone bool
}

var ErrorPages = []ErrorPage{
	{Code: "400", Title: "Bad request", Message: "The request could not be understood. Check the address and try again.", Standalone: true},
	{Code: "403", Title: "Forbidden", Message: "You do not have permission to view this page."},
	{Code: "404", Title: "Not found", Message: "There is nothing at this address."},
	{Code: "500", Title: "Server error", Message: "Something went wrong on our end. The error has been recorded.", Standalone: true},
}

func ErrorTemplate(page ErrorPage, tailwind bool) string {
	switch {
	case page.Standalone:
		return Render("error_standalone.html.tmpl", page)
	case tailwind:
		return Render("error_tailwind.html.tmpl", page)
	default:
		return Render("error_plain.html.tmpl", page)
	}
}
