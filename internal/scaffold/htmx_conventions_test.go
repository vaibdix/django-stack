package scaffold

import (
	"regexp"
	"strings"
	"testing"

	"github.com/vaibdix/django-cli/internal/assets"
	"github.com/vaibdix/django-cli/internal/config"
)

func appTemplates(t *testing.T, tailwind, htmx bool) map[string]string {
	t.Helper()

	cfg := AppTemplateConfig{AppName: "catalog", Tailwind: tailwind, HTMX: htmx}
	out := map[string]string{
		"book_list.html":   generateBookListTemplate(cfg),
		"book_form.html":   generateBookFormTemplate(cfg),
		"book_delete.html": generateBookDeleteTemplate(cfg),
	}
	if htmx {
		out["partials/book_list_items.html"] = generateBookListItemsPartial(cfg)
		out["partials/book_form_partial.html"] = generateBookFormPartial(cfg)
	}
	return out
}

var hardcodedAppPath = regexp.MustCompile(`(href|hx-get|hx-post|hx-put|hx-delete)="/catalog/`)

func TestGeneratedTemplatesReverseURLsInsteadOfHardcodingThem(t *testing.T) {
	t.Parallel()

	for _, tailwind := range []bool{false, true} {
		for _, htmx := range []bool{false, true} {
			for name, body := range appTemplates(t, tailwind, htmx) {
				if match := hardcodedAppPath.FindString(body); match != "" {
					t.Errorf("tailwind=%v htmx=%v %s hardcodes an app path (%s); "+
						"use {%% url %%} so the app survives being mounted elsewhere",
						tailwind, htmx, name, match)
				}
			}
		}
	}
}

func TestEveryHTMXRequestAttributeNamesItsURL(t *testing.T) {
	t.Parallel()

	// `hx-post` with no value posts to whatever URL the fragment came from.
	// That happens to work for the create form and silently breaks anywhere else.
	bare := regexp.MustCompile(`hx-(get|post|put|patch|delete)(\s|>)`)

	for _, tailwind := range []bool{false, true} {
		for name, body := range appTemplates(t, tailwind, true) {
			if match := bare.FindString(body); match != "" {
				t.Errorf("tailwind=%v %s has a valueless %q", tailwind, name, strings.TrimSpace(match))
			}
		}
	}
}

func TestHTMXTemplatesKeepTheirSwapContract(t *testing.T) {
	t.Parallel()

	for _, tailwind := range []bool{false, true} {
		templates := appTemplates(t, tailwind, true)

		list := templates["book_list.html"]
		for _, needle := range []string{
			`id="book-list"`,
			`hx-target="#book-list"`,
			`hx-trigger="keyup changed delay:300ms, search"`,
			`hx-push-url="true"`,
		} {
			if !strings.Contains(list, needle) {
				t.Errorf("tailwind=%v book_list.html is missing %s", tailwind, needle)
			}
		}

		items := templates["partials/book_list_items.html"]
		for _, needle := range []string{
			`id="book-{{ book.id }}"`,
			`hx-target="closest article"`,
			`hx-swap="outerHTML"`,
			`hx-confirm=`,
		} {
			if !strings.Contains(items, needle) {
				t.Errorf("tailwind=%v the list partial is missing %s", tailwind, needle)
			}
		}
	}
}

func TestSearchRendersAWholePageWhenItIsNotAnHTMXRequest(t *testing.T) {
	t.Parallel()

	views := generateBookViews("catalog", true, false)

	idx := strings.Index(views, "def book_search")
	if idx == -1 {
		t.Fatal("the HTMX variant generates no search view")
	}
	search := views[idx:]

	if !strings.Contains(search, "if wants_fragment(request):") {
		t.Errorf("book_search returns a bare fragment to every caller, so opening the search "+
			"URL directly shows unstyled markup:\n%s", search)
	}
	if !strings.Contains(search, "book_list.html") {
		t.Errorf("book_search has no full-page branch:\n%s", search)
	}
}

func TestNonHTMXProjectsGetNoHTMXAttributes(t *testing.T) {
	t.Parallel()

	for _, tailwind := range []bool{false, true} {
		for name, body := range appTemplates(t, tailwind, false) {
			if strings.Contains(body, "hx-") {
				t.Errorf("tailwind=%v %s carries HTMX attributes in a non-HTMX project", tailwind, name)
			}
		}
	}
}

func TestProjectNameThatWouldShadowAnInstalledModuleIsRejected(t *testing.T) {
	t.Parallel()

	// The reported failure: a project called django-htmx shadows the django_htmx
	// package, so MIDDLEWARE cannot import django_htmx.middleware.
	for _, name := range []string{"django-htmx", "django_htmx", "django", "rest-framework", "mypy"} {
		if err := (config.Project{ProjectName: name}).ValidateProjectName(); err == nil {
			t.Errorf("%q was accepted; it shadows an installed package and the project cannot boot", name)
		}
	}

	for _, name := range []string{"bookshop", "django-stack", "my-htmx-app", "htmx-demo"} {
		if err := (config.Project{ProjectName: name}).ValidateProjectName(); err != nil {
			t.Errorf("%q is a fine project name but was rejected: %v", name, err)
		}
	}
}

func TestBoostedNavigationGetsTheWholePageNotAFragment(t *testing.T) {
	t.Parallel()

	views := generateBookViews("catalog", true, false)

	// hx-boost sets HX-Request on ordinary navigation, so `request.htmx` alone is
	// true for a plain link click. Branching on it swaps a bare fragment into
	// <body> and the page loses its chrome and its stylesheet.
	if strings.Contains(views, "if request.htmx:") ||
		strings.Contains(views, "if request.htmx else") {
		t.Errorf("a view branches on request.htmx without excluding boosted navigation:\n%s", views)
	}
	if !strings.Contains(views, "not request.htmx.boosted") {
		t.Errorf("no view distinguishes a boosted request from a partial one:\n%s", views)
	}

	for _, name := range []string{"book_list", "book_create", "book_edit", "book_delete", "book_search"} {
		idx := strings.Index(views, "def "+name)
		if idx == -1 {
			t.Errorf("%s is missing from the HTMX views", name)
		}
	}
}

func TestTheCrudStylesheetIsLinkedFromTheBaseTemplate(t *testing.T) {
	t.Parallel()

	// hx-boost swaps <body> only, so anything the CRUD pages need in <head> has
	// to come from base.html or it vanishes on boosted navigation.
	base := assets.RenderPage(assets.PageBase, assets.PageOptions{
		AppName: "catalog", HTMX: true, GlobalTemplates: true,
	})
	if !strings.Contains(base, "css/app.css") {
		t.Errorf("base.html does not link the CRUD stylesheet:\n%s", base)
	}

	for _, tailwind := range []bool{false, true} {
		for name, body := range appTemplates(t, tailwind, true) {
			if strings.Contains(body, "css/app.css") {
				t.Errorf("tailwind=%v %s links app.css itself; a boosted swap would drop it",
					tailwind, name)
			}
		}
	}
}
