package assets

import "fmt"

type ErrorPage struct {
	Code    string
	Title   string
	Message string

	// Standalone pages extend nothing and use no tags. Django renders 400.html
	// and 500.html with an empty Context and no request, so that a failure in
	// a context processor, the static manifest or the database cannot take
	// the error page down with it.
	Standalone bool
}

var ErrorPages = []ErrorPage{
	{Code: "400", Title: "Bad request", Message: "The request could not be understood. Check the address and try again.", Standalone: true},
	{Code: "403", Title: "Forbidden", Message: "You do not have permission to view this page."},
	{Code: "404", Title: "Not found", Message: "There is nothing at this address."},
	{Code: "500", Title: "Server error", Message: "Something went wrong on our end. The error has been recorded.", Standalone: true},
}

func ErrorTemplate(page ErrorPage, tailwind bool) string {
	if page.Standalone {
		return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <meta name="description" content="%s">
    <title>%s — %s</title>
</head>
<body>
    <main>
        <h1>%s</h1>
        <p>%s</p>
        <p><a href="/">Back to home</a></p>
    </main>
</body>
</html>
`, page.Message, page.Code, page.Title, page.Title, page.Message)
	}
	if tailwind {
		return fmt.Sprintf(`{%% extends "base.html" %%}
{%% block title %%}%s — %s{%% endblock %%}
{%% block content %%}
<section class="flex min-h-[50vh] items-center justify-center px-5 py-10">
    <div class="panel w-full max-w-[560px] p-8 text-center">
        <code>%s</code>
        <h1 class="mt-4 mb-2">%s</h1>
        <p class="text-body">%s</p>
        <a href="{%% url 'home' %%}" class="button button-primary mt-6 inline-block">Back to home</a>
    </div>
</section>
{%% endblock %%}
`, page.Code, page.Title, page.Code, page.Title, page.Message)
	}

	return fmt.Sprintf(`{%% extends "base.html" %%}
{%% block title %%}%s — %s{%% endblock %%}
{%% block content %%}
<section class="error-page">
    <div class="error-card">
        <code>%s</code>
        <h1>%s</h1>
        <p>%s</p>
        <a href="{%% url 'home' %%}" class="button button-primary">Back to home</a>
    </div>
</section>
{%% endblock %%}
`, page.Code, page.Title, page.Code, page.Title, page.Message)
}
