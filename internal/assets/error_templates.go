package assets

import "fmt"

type ErrorPage struct {
	Code    string
	Title   string
	Message string
}

var ErrorPages = []ErrorPage{
	{Code: "400", Title: "Bad request", Message: "The request could not be understood. Check the address and try again."},
	{Code: "403", Title: "Forbidden", Message: "You do not have permission to view this page."},
	{Code: "404", Title: "Not found", Message: "There is nothing at this address."},
	{Code: "500", Title: "Server error", Message: "Something went wrong on our end. The error has been recorded."},
}

func ErrorTemplate(page ErrorPage, tailwind bool) string {
	if tailwind {
		return fmt.Sprintf(`{%% extends 'base.html' %%}
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

	return fmt.Sprintf(`{%% extends 'base.html' %%}
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
