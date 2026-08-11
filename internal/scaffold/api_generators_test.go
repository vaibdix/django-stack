package scaffold

import (
	"strings"
	"testing"
)

func TestGenerateBookModelContainsBookClass(t *testing.T) {
	t.Parallel()

	model := generateBookModel()

	required := []string{
		"class Book(models.Model):",
		"title = models.CharField",
		"author = models.CharField",
		"isbn = models.CharField",
		"price = models.DecimalField",
		"def __str__(self):",
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
	if !strings.Contains(s, "fields = '__all__'") {
		t.Fatalf("expected fields = '__all__', got:\n%s", s)
	}
}

func TestGenerateBookFormsWithTailwind(t *testing.T) {
	t.Parallel()

	plain := generateBookForms(false)
	tw := generateBookForms(true)

	if !strings.Contains(plain, "class BookForm(forms.ModelForm):") {
		t.Fatalf("expected BookForm class, got:\n%s", plain)
	}

	if !strings.Contains(tw, "w-full rounded-2xl border border-white/10") {
		t.Fatalf("expected tailwind classes in tailwind form, got:\n%s", tw)
	}
	if strings.Contains(plain, "w-full rounded-2xl") {
		t.Fatalf("plain form should not have tailwind classes, got:\n%s", plain)
	}
}

func TestGenerateBookAPIRegistersRouter(t *testing.T) {
	t.Parallel()

	api := generateBookAPI()

	if !strings.Contains(api, "router.register(r'books', BookViewSet") {
		t.Fatalf("expected router registration, got:\n%s", api)
	}
	if !strings.Contains(api, "urlpatterns = router.urls") {
		t.Fatalf("expected urlpatterns assignment, got:\n%s", api)
	}
}

func TestGenerateBookViewsWithHTMX(t *testing.T) {
	t.Parallel()

	plain := generateBookViews("library", false)
	htmx := generateBookViews("library", true)

	if strings.Contains(plain, "request.htmx") {
		t.Fatalf("plain views should not reference request.htmx, got:\n%s", plain)
	}
	if !strings.Contains(htmx, "request.htmx") {
		t.Fatalf("htmx views should reference request.htmx, got:\n%s", htmx)
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

	admin := generateAdminPy()

	required := []string{
		"from .models import Book",
		"@admin.register(Book)",
		"class BookAdmin(admin.ModelAdmin):",
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

	data := generateSampleData("books")

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
