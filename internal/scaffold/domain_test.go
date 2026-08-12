package scaffold

import (
	"strings"
	"testing"

	"github.com/vaibdix/django-cli/internal/config"
)

func TestBookQueriesAreDefinedOnceOnTheModel(t *testing.T) {
	t.Parallel()

	model := generateBookModel("catalog", true, "6.1", config.SQLite)

	for _, want := range []string{
		"class BookQuerySet(models.QuerySet['Book']):",
		"def search(self, term: str) -> 'BookQuerySet':",
		"def recent(self, days: int = RECENT_WINDOW_DAYS) -> 'BookQuerySet':",
		"class BookManager(models.Manager['Book']):",
		"objects = BookManager()",
	} {
		if !strings.Contains(model, want) {
			t.Errorf("models.py is missing %q, so the queries live in the callers instead:\n%s", want, model)
		}
	}
}

func TestBothCallersGoThroughTheModelQueries(t *testing.T) {
	t.Parallel()

	api := generateBookAPI(false)
	if !strings.Contains(api, "return Book.objects.search(self.request.query_params.get('search', ''))") {
		t.Errorf("the API filters for itself instead of asking the model:\n%s", api)
	}
	if !strings.Contains(api, "self.get_queryset().recent()") {
		t.Errorf("the recent action re-implements its own window:\n%s", api)
	}
	if strings.Contains(api, "search_fields") {
		t.Errorf("the API still declares a second definition of what a search matches:\n%s", api)
	}

	views := generateBookViews("catalog", true, false, true)
	if !strings.Contains(views, "Book.objects.search(") {
		t.Errorf("the HTML views filter for themselves instead of asking the model:\n%s", views)
	}
	if strings.Contains(views, "title__icontains") {
		t.Errorf("the HTML search still matches on title alone, so it disagrees with the API:\n%s", views)
	}
}

func TestModelConstraintsUseTheKeywordTheChosenDjangoAccepts(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		version string
		keyword string
		absent  string
	}{
		{version: "6.1", keyword: "condition=Q(price__gte=0)", absent: "check=Q("},
		{version: "5.1.6", keyword: "condition=Q(price__gte=0)", absent: "check=Q("},
		{version: "5.0.14", keyword: "check=Q(price__gte=0)", absent: "condition=Q("},
	} {
		t.Run(tc.version, func(t *testing.T) {
			t.Parallel()

			model := generateBookModel("catalog", true, tc.version, config.SQLite)
			if !strings.Contains(model, tc.keyword) {
				t.Errorf("Django %s needs %q:\n%s", tc.version, tc.keyword, model)
			}
			if strings.Contains(model, tc.absent) {
				t.Errorf("Django %s does not accept %q:\n%s", tc.version, tc.absent, model)
			}
		})
	}
}

func TestPriceAndDateInvariantsReachTheDatabase(t *testing.T) {
	t.Parallel()

	model := generateBookModel("catalog", true, "6.1", config.SQLite)

	for _, want := range []string{
		"constraints = [",
		"name='book_price_not_negative'",
		"name='book_publication_date_plausible'",
	} {
		if !strings.Contains(model, want) {
			t.Errorf("models.py leaves %q to the form layer, so ORM writes bypass it:\n%s", want, model)
		}
	}
}

func TestOrderingIsTotalSoPaginationCannotRepeatARow(t *testing.T) {
	t.Parallel()

	model := generateBookModel("catalog", true, "6.1", config.SQLite)
	if !strings.Contains(model, "ordering = ('-created_at', '-pk')") {
		t.Errorf("created_at is not unique, so ordering by it alone lets pagination skip or repeat:\n%s", model)
	}
}

func TestAbsoluteURLOnlyExistsWhereTheRouteDoes(t *testing.T) {
	t.Parallel()

	if withViews := generateBookModel("catalog", true, "6.1", config.SQLite); !strings.Contains(withViews, "def get_absolute_url(self) -> str:") {
		t.Errorf("a project with HTML routes should give the model its URL:\n%s", withViews)
	}
	apiOnly := generateBookModel("catalog", false, "6.1", config.SQLite)
	if strings.Contains(apiOnly, "get_absolute_url") {
		t.Errorf("an API-only project has no book_edit route, so reversing it would raise:\n%s", apiOnly)
	}
	if strings.Contains(apiOnly, "from django.urls import reverse") {
		t.Errorf("an API-only project imports reverse and never uses it:\n%s", apiOnly)
	}
}

func TestTitleCarriesNoIndexThatCannotServeItsQuery(t *testing.T) {
	t.Parallel()

	model := generateBookModel("catalog", true, "6.1", config.SQLite)

	if strings.Contains(model, "db_index=True") {
		t.Errorf("the search is icontains, which a B-tree cannot serve, so the index is write cost for nothing:\n%s", model)
	}
}
