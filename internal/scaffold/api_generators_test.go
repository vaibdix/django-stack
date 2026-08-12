package scaffold

import (
	"strings"
	"testing"
)

func TestGenerateBookModelContainsBookClass(t *testing.T) {
	t.Parallel()

	model := generateBookModel("catalog", true, "6.1")

	required := []string{
		"class Book(models.Model):",
		"title = models.CharField",
		"author = models.CharField",
		"isbn = models.CharField",
		"price = models.DecimalField",
		"def __str__(self) -> str:",
		"ordering = ('-created_at', '-pk')",
	}
	for _, frag := range required {
		if !strings.Contains(model, frag) {
			t.Fatalf("expected %q in book model, got:\n%s", frag, model)
		}
	}
}

func TestGenerateBookSerializerReferencesBook(t *testing.T) {
	t.Parallel()

	s := generateBookSerializer()

	if !strings.Contains(s, "from .models import Book") {
		t.Fatalf("expected Book import, got:\n%s", s)
	}
	if !strings.Contains(s, "class BookSerializer") {
		t.Fatalf("expected BookSerializer class, got:\n%s", s)
	}
	if strings.Contains(s, "fields = '__all__'") {
		t.Fatalf("the serializer exposes every column, so a field added later is published without review:\n%s", s)
	}
	for _, field := range []string{"'title'", "'author'", "'isbn'", "'publication_date'", "'price'"} {
		if !strings.Contains(s, field) {
			t.Fatalf("expected %s to be listed explicitly, got:\n%s", field, s)
		}
	}
	if !strings.Contains(s, "read_only_fields = ('id', 'created_at', 'updated_at')") {
		t.Fatalf("expected the timestamps to stay read-only, got:\n%s", s)
	}
}

func TestGenerateBookFormsWithTailwind(t *testing.T) {
	t.Parallel()

	plain := generateBookForms(false, true)
	tw := generateBookForms(true, true)

	if !strings.Contains(plain, "class BookForm(forms.ModelForm[Book]):") {
		t.Fatalf("expected BookForm class, got:\n%s", plain)
	}

	if !strings.Contains(tw, "border border-rule bg-ground") {
		t.Fatalf("expected the widget classes to name design tokens, got:\n%s", tw)
	}
	if strings.Contains(tw, "white/") {
		t.Fatalf("the tailwind form still styles with opacity literals rather than tokens, got:\n%s", tw)
	}
	if strings.Contains(plain, "border-rule") {
		t.Fatalf("plain form should not have tailwind classes, got:\n%s", plain)
	}
}

func TestGenerateBookAPIRegistersRouter(t *testing.T) {
	t.Parallel()

	api := generateBookAPI(false)

	if !strings.Contains(api, "router.register(r'books', BookViewSet") {
		t.Fatalf("expected router registration, got:\n%s", api)
	}
	if !strings.Contains(api, "urlpatterns = router.urls") {
		t.Fatalf("expected urlpatterns assignment, got:\n%s", api)
	}
}

func TestGenerateBookViewsWithHTMX(t *testing.T) {
	t.Parallel()

	plain := generateBookViews("library", false, false)
	htmx := generateBookViews("library", true, false)

	if strings.Contains(plain, "wants_fragment") {
		t.Fatalf("plain views should not branch on htmx, got:\n%s", plain)
	}
	if !strings.Contains(htmx, "wants_fragment(request)") {
		t.Fatalf("htmx views should branch on wants_fragment, got:\n%s", htmx)
	}
	if !strings.Contains(generateHtmxSupport(), "request.htmx") {
		t.Fatalf("htmx.py should own the request.htmx check, got:\n%s", generateHtmxSupport())
	}
	if !strings.Contains(htmx, "book_search") {
		t.Fatalf("htmx views should include book_search, got:\n%s", htmx)
	}
}

func TestGenerateAppURLsWithHTMX(t *testing.T) {
	t.Parallel()

	plain := generateAppURLs("library", false)
	htmx := generateAppURLs("library", true)

	if !strings.Contains(plain, "app_name = 'library'") {
		t.Fatalf("expected app_name, got:\n%s", plain)
	}
	if strings.Contains(plain, "book_search") {
		t.Fatalf("plain URLs should not include search, got:\n%s", plain)
	}
	if !strings.Contains(htmx, "book_search") {
		t.Fatalf("htmx URLs should include search, got:\n%s", htmx)
	}
}

func TestGenerateAdminPyRegistersBook(t *testing.T) {
	t.Parallel()

	admin := generateAdminPy(true)

	required := []string{
		"from .models import Book",
		"@admin.register(Book)",
		"class BookAdmin(admin.ModelAdmin[Book]):",
		"list_display",
		"search_fields",
		"readonly_fields",
	}
	for _, frag := range required {
		if !strings.Contains(admin, frag) {
			t.Fatalf("expected %q in admin.py, got:\n%s", frag, admin)
		}
	}
}

func TestGenerateSampleDataReferencesAppName(t *testing.T) {
	t.Parallel()

	data := generateSampleData("books", false)

	if !strings.Contains(data, "from books.models import Book") {
		t.Fatalf("expected app-specific import, got:\n%s", data)
	}
	if !strings.Contains(data, "class Command(BaseCommand):") {
		t.Fatalf("expected Command class, got:\n%s", data)
	}
	if !strings.Contains(data, "update_or_create") {
		t.Fatalf("expected update_or_create for idempotency, got:\n%s", data)
	}
}
