package scaffold

import (
	"strings"
	"testing"

	"github.com/vaibdix/django-stack/internal/config"
)

func TestBookQueriesAreDefinedOnceOnTheModel(t *testing.T) {
	t.Parallel()

	model := generateBookModel("catalog", true)
	if strings.Contains(model, "class BookManager") {
		t.Errorf("a hand-written manager re-declares every queryset method; as_manager() copies them:\n%s", model)
	}

	for _, want := range []string{
		"class BookQuerySet(models.QuerySet['Book']):",
		"def search(self, term: str) -> 'BookQuerySet':",
		"def recent(self, days: int = RECENT_WINDOW_DAYS) -> 'BookQuerySet':",
		"objects = BookQuerySet.as_manager()",
	} {
		if !strings.Contains(model, want) {
			t.Errorf("models.py is missing %q, so the queries live in the callers instead:\n%s", want, model)
		}
	}
}

func TestBothCallersGoThroughTheModelQueries(t *testing.T) {
	t.Parallel()

	api := generateBookAPI()
	if !strings.Contains(api, "return Book.objects.search(self.request.query_params.get('search', ''))") {
		t.Errorf("the API filters for itself instead of asking the model:\n%s", api)
	}
	if !strings.Contains(api, "self.get_queryset().recent()") {
		t.Errorf("the recent action re-implements its own window:\n%s", api)
	}
	if strings.Contains(api, "search_fields") {
		t.Errorf("the API still declares a second definition of what a search matches:\n%s", api)
	}

	views := renderBookViews(bookViewsData{AppName: "catalog", HTMX: true, RuntimeGenerics: true})
	if !strings.Contains(views, "Book.objects.search(") {
		t.Errorf("the HTML views filter for themselves instead of asking the model:\n%s", views)
	}
	if strings.Contains(views, "title__icontains") {
		t.Errorf("the HTML search still matches on title alone, so it disagrees with the API:\n%s", views)
	}
}

func TestModelConstraintsUseTheConditionKeyword(t *testing.T) {
	t.Parallel()

	model := generateBookModel("catalog", true)
	if !strings.Contains(model, "condition=Q(price__gte=0)") || strings.Contains(model, "check=Q(") {
		t.Errorf("every supported Django takes condition= on CheckConstraint:\n%s", model)
	}
}

func TestPriceAndDateInvariantsReachTheDatabase(t *testing.T) {
	t.Parallel()

	model := generateBookModel("catalog", true)

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

	model := generateBookModel("catalog", true)
	if !strings.Contains(model, "ordering = ('-created_at', '-pk')") {
		t.Errorf("created_at is not unique, so ordering by it alone lets pagination skip or repeat:\n%s", model)
	}
}

func TestAbsoluteURLOnlyExistsWhereTheRouteDoes(t *testing.T) {
	t.Parallel()

	if withViews := generateBookModel("catalog", true); !strings.Contains(withViews, "def get_absolute_url(self) -> str:") {
		t.Errorf("a project with HTML routes should give the model its URL:\n%s", withViews)
	}
	apiOnly := generateBookModel("catalog", false)
	if strings.Contains(apiOnly, "get_absolute_url") {
		t.Errorf("an API-only project has no book_edit route, so reversing it would raise:\n%s", apiOnly)
	}
	if strings.Contains(apiOnly, "from django.urls import reverse") {
		t.Errorf("an API-only project imports reverse and never uses it:\n%s", apiOnly)
	}
}

func TestTitleCarriesNoIndexThatCannotServeItsQuery(t *testing.T) {
	t.Parallel()

	model := generateBookModel("catalog", true)

	if strings.Contains(model, "db_index=True") {
		t.Errorf("the search is icontains, which a B-tree cannot serve, so the index is write cost for nothing:\n%s", model)
	}
}

func TestTimestampsComeFromOneSharedBase(t *testing.T) {
	t.Parallel()

	model := generateBookModel("catalog", true)

	if !strings.Contains(model, "class Book(TimeStampedModel):") {
		t.Errorf("the model redeclares its own timestamps:\n%s", model)
	}
	if strings.Contains(model, "created_at = models.DateTimeField") {
		t.Errorf("created_at is declared twice, once here and once in the shared base:\n%s", model)
	}

	base := renderStatic("core_models.py.tmpl")
	for _, want := range []string{"class TimeStampedModel(models.Model):", "abstract = True", "get_latest_by = 'created_at'"} {
		if !strings.Contains(base, want) {
			t.Errorf("core/models.py is missing %q:\n%s", want, base)
		}
	}
}

func TestSeedDataUpsertsInOneStatement(t *testing.T) {
	t.Parallel()

	seed := generateSampleData("catalog", config.SQLite)

	for _, want := range []string{"bulk_create(", "update_conflicts=True", "unique_fields=['isbn']"} {
		if !strings.Contains(seed, want) {
			t.Errorf("the seed command is missing %q, so it issues a query per row:\n%s", want, seed)
		}
	}
}

func TestTheSeedNamesAConflictTargetOnlyWhereTheBackendTakesOne(t *testing.T) {
	t.Parallel()

	for _, db := range []config.Database{
		config.SQLite, config.PostgresDocker, config.MySQLDocker,
	} {
		seed := generateSampleData("catalog", db)

		if !strings.Contains(seed, "update_conflicts=True") {
			t.Errorf("%v seed does not upsert, so re-running it fails on the second call:\n%s", db, seed)
		}

		named := strings.Contains(seed, "unique_fields=['isbn']")
		if named != db.SupportsUpsertTarget() {
			t.Errorf("%v seed names a conflict target = %v, want %v; MySQL raises "+
				"NotSupportedError when one is given, and every other backend raises "+
				"ValueError when one is missing:\n%s", db, named, db.SupportsUpsertTarget(), seed)
		}
	}
}

func TestTimestampsDoNotRestateWhatAutoNowImplies(t *testing.T) {
	t.Parallel()

	if base := renderStatic("core_models.py.tmpl"); strings.Contains(base, "editable=False") {
		t.Errorf("editable=False restates what auto_now/auto_now_add already imply:\n%s", base)
	}
}

func TestTheSeedDoesNotWrapItsOneStatementInATransaction(t *testing.T) {
	t.Parallel()

	if seed := generateSampleData("catalog", config.SQLite); strings.Contains(seed, "transaction.atomic") {
		t.Errorf("a single bulk_create is already atomic:\n%s", seed)
	}
}
