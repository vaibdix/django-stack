package scaffold

import (
	"strings"
	"testing"
)

func TestEveryMutatingHTMLViewRequiresALogin(t *testing.T) {
	t.Parallel()

	for _, htmx := range []bool{false, true} {
		views := generateBookViews("catalog", htmx, true)

		if !strings.Contains(views, "from django.contrib.auth.mixins import LoginRequiredMixin") {
			t.Fatalf("htmx=%v the views do not import LoginRequiredMixin:\n%s", htmx, views)
		}

		for _, view := range []string{"BookCreateView", "BookUpdateView", "BookDeleteView"} {
			idx := strings.Index(views, "class "+view+"(")
			if idx == -1 {
				t.Fatalf("htmx=%v %s is missing", htmx, view)
			}
			end := strings.Index(views[idx:], "\n)")
			if end == -1 {
				end = strings.Index(views[idx:], ":\n")
			}
			if !strings.Contains(views[idx:idx+end], "LoginRequiredMixin") {
				t.Errorf("htmx=%v %s can be written to anonymously while the API rejects the same call:\n%s",
					htmx, view, views[idx:idx+end])
			}
		}

		if strings.Contains(views, "class BookListView(LoginRequiredMixin") {
			t.Errorf("htmx=%v reading the catalog should not require an account:\n%s", htmx, views)
		}
	}
}

func TestFormErrorsAreRenderedInEveryVariant(t *testing.T) {
	t.Parallel()

	for _, tailwind := range []bool{false, true} {
		fields := generateBookFieldsPartial(AppTemplateConfig{AppName: "catalog", Tailwind: tailwind})

		for _, want := range []string{
			"{% for error in form.non_field_errors %}",
			"{% for field in form %}",
			"{% for error in field.errors %}",
			"{{ field.label_tag }}",
			"{% if field.help_text %}",
		} {
			if !strings.Contains(fields, want) {
				t.Errorf("tailwind=%v the field partial is missing %q, so a rejected submission "+
					"returns 200 with nothing to read:\n%s", tailwind, want, fields)
			}
		}
	}
}

func TestFormPagesRenderFieldsThroughTheOnePartial(t *testing.T) {
	t.Parallel()

	for _, tailwind := range []bool{false, true} {
		for _, htmx := range []bool{false, true} {
			cfg := AppTemplateConfig{AppName: "catalog", Tailwind: tailwind, HTMX: htmx}

			page := generateBookFormTemplate(cfg)
			if !strings.Contains(page, "{% include \"catalog/partials/book_fields.html\" %}") {
				t.Errorf("tailwind=%v htmx=%v the form page hand-writes its fields:\n%s", tailwind, htmx, page)
			}
			if strings.Contains(page, "{{ form.title }}") {
				t.Errorf("tailwind=%v htmx=%v the form page still names fields one by one:\n%s", tailwind, htmx, page)
			}

			list := generateBookListTemplate(cfg)
			if !strings.Contains(list, "{% include \"catalog/partials/book_list_items.html\" %}") {
				t.Errorf("tailwind=%v htmx=%v the list page duplicates its own card markup:\n%s", tailwind, htmx, list)
			}
		}
	}
}

func TestTheListTargetSurvivesAnEmptyResult(t *testing.T) {
	t.Parallel()

	for _, tailwind := range []bool{false, true} {
		list := generateBookListTemplate(AppTemplateConfig{AppName: "catalog", Tailwind: tailwind, HTMX: true})

		idx := strings.Index(list, `id="book-list"`)
		if idx == -1 {
			t.Fatalf("tailwind=%v the list page has no swap target:\n%s", tailwind, list)
		}
		if strings.Contains(list[:idx], "{% if books %}") {
			t.Errorf("tailwind=%v the swap target is inside the non-empty branch, so search "+
				"cannot recover from an empty result:\n%s", tailwind, list)
		}
		if !strings.Contains(list, `value="{{ query }}"`) {
			t.Errorf("tailwind=%v the search box forgets the term it searched for:\n%s", tailwind, list)
		}
	}
}

func TestTheHTMLListIsPaginated(t *testing.T) {
	t.Parallel()

	views := generateBookViews("catalog", true, true)
	if !strings.Contains(views, "paginate_by = PAGE_SIZE") {
		t.Errorf("the HTML list is unbounded while the API paginates at 20:\n%s", views)
	}

	for _, tailwind := range []bool{false, true} {
		items := generateBookListItemsPartial(AppTemplateConfig{AppName: "catalog", Tailwind: tailwind, HTMX: true})
		if !strings.Contains(items, "{% if page_obj.has_other_pages %}") {
			t.Errorf("tailwind=%v nothing renders the pages the view produces:\n%s", tailwind, items)
		}
	}
}

func TestRejectedHTMXSubmissionsDoNotReadAsSuccess(t *testing.T) {
	t.Parallel()

	views := generateBookViews("catalog", true, true)
	if !strings.Contains(views, "HTTPStatus.UNPROCESSABLE_ENTITY") {
		t.Errorf("an invalid htmx post still returns 200, so htmx swaps a blank form over the list:\n%s", views)
	}

	plain := generateBookViews("catalog", false, true)
	if strings.Contains(plain, "UNPROCESSABLE_ENTITY") {
		t.Errorf("a non-htmx project has no client to interpret a 422:\n%s", plain)
	}
}

func TestASuccessfulFragmentSubmitHandsControlBackToTheBrowser(t *testing.T) {
	t.Parallel()

	views := generateBookViews("catalog", true, true)

	if !strings.Contains(views, "response['HX-Redirect'] = url") {
		t.Errorf("a successful htmx submit returns a 302 that htmx follows and swaps into the form "+
			"container, replacing the form with a whole page:\n%s", views)
	}
	if !strings.Contains(views, "HTTPStatus.NO_CONTENT") {
		t.Errorf("the redirect response still carries a body htmx would try to swap:\n%s", views)
	}

	plain := generateBookViews("catalog", false, true)
	if strings.Contains(plain, "HX-Redirect") {
		t.Errorf("a non-htmx project has no client to read that header:\n%s", plain)
	}
}
