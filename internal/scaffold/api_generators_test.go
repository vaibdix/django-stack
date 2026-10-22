package scaffold

import (
	"strings"
	"testing"

	"github.com/vaibdix/django-stack/internal/config"
)

func TestGenerateBookModelContainsBookClass(t *testing.T) {
	t.Parallel()

	model := generateBookModel("catalog", true, config.SQLite)

	required := []string{
		"class Book(TimeStampedModel):",
		"from core.models import TimeStampedModel",
		"title = models.CharField",
		"author = models.CharField",
		"isbn = ISBNField(",
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

	s := generateAPIPackage("catalog")["serializers.py"]

	if !strings.Contains(s, "from catalog.models import Book") {
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

	plain := generateBookForms(bookFormsData{Tailwind: false, RuntimeGenerics: true})
	tw := generateBookForms(bookFormsData{Tailwind: true, RuntimeGenerics: true})

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

	api := generateBookAPI()

	if !strings.Contains(api, "class BookViewSet(viewsets.ModelViewSet[Book]):") {
		t.Fatalf("expected the viewset, got:\n%s", api)
	}

	pkg := generateAPIPackage("catalog")
	if !strings.Contains(pkg["urls.py"], "router.register(r'books', BookViewSet") {
		t.Fatalf("expected router registration, got:\n%s", pkg["urls.py"])
	}
	if !strings.Contains(pkg["urls.py"], "urlpatterns = router.urls") {
		t.Fatalf("expected urlpatterns assignment, got:\n%s", pkg["urls.py"])
	}
	if strings.Contains(api, "router") {
		t.Fatalf("the viewset module still owns the URLconf:\n%s", api)
	}
	for _, name := range []string{"__init__.py", "views.py", "serializers.py", "urls.py"} {
		if pkg[name] == "" {
			t.Fatalf("the api package is missing %s", name)
		}
	}
}

func TestGenerateBookViewsWithHTMX(t *testing.T) {
	t.Parallel()

	plain := renderBookViews(bookViewsData{AppName: "library", HTMX: false, RuntimeGenerics: true})
	htmx := renderBookViews(bookViewsData{AppName: "library", HTMX: true, RuntimeGenerics: true})

	if strings.Contains(plain, "wants_fragment") {
		t.Fatalf("plain views should not branch on htmx, got:\n%s", plain)
	}
	if strings.Contains(plain, "HtmxTemplateMixin") {
		t.Fatalf("plain views should not mix in a module their project does not generate, got:\n%s", plain)
	}
	if !strings.Contains(htmx, "HtmxTemplateMixin") {
		t.Fatalf("htmx views should pick their template through the mixin, got:\n%s", htmx)
	}
	if !strings.Contains(generateHtmxSupport(), ".htmx") {
		t.Fatalf("htmx.py should own the request.htmx check, got:\n%s", generateHtmxSupport())
	}
}

func TestGenerateAppURLsRouteToClassBasedViews(t *testing.T) {
	t.Parallel()

	urls := generateAppURLs("library")

	if !strings.Contains(urls, "app_name = 'library'") {
		t.Fatalf("expected app_name, got:\n%s", urls)
	}
	for _, want := range []string{
		"views.BookListView.as_view()",
		"views.BookCreateView.as_view()",
		"views.BookUpdateView.as_view()",
		"views.BookDeleteView.as_view()",
	} {
		if !strings.Contains(urls, want) {
			t.Fatalf("expected %q, got:\n%s", want, urls)
		}
	}
	if strings.Contains(urls, "book_search") {
		t.Fatalf("search is the list view now, so a second route would be a second definition:\n%s", urls)
	}
}

func TestGenerateAdminPyRegistersBook(t *testing.T) {
	t.Parallel()

	admin := generateAdminPy(typingData{RuntimeGenerics: true})

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

	data := generateSampleData("books", config.SQLite)

	if !strings.Contains(data, "from books.models import Book") {
		t.Fatalf("expected app-specific import, got:\n%s", data)
	}
	if !strings.Contains(data, "class Command(BaseCommand):") {
		t.Fatalf("expected Command class, got:\n%s", data)
	}
	if !strings.Contains(data, "update_conflicts=True") {
		t.Fatalf("expected a single upserting bulk_create for idempotency, got:\n%s", data)
	}
	if strings.Contains(data, "update_or_create") {
		t.Fatalf("the seed command still issues one query per row:\n%s", data)
	}
}

func generateBookAPI() string {
	return generateAPIPackage("catalog")["views.py"]
}

func TestEveryAppRegistersTheBookAdmin(t *testing.T) {
	t.Parallel()

	content := generateAdminPy(typingData{RuntimeGenerics: false})
	for _, fragment := range []string{
		"from .models import Book",
		"@admin.register(Book)",
		"list_display = (\"title\", \"author\", \"isbn\", \"publication_date\", \"price\", \"created_at\")",
		"readonly_fields = (\"created_at\", \"updated_at\")",
	} {
		if !strings.Contains(content, fragment) {
			t.Fatalf("admin.py is missing %q:\n%s", fragment, content)
		}
	}
}

func TestGeneratedBookViewsUseGetObjectOr404ForStaleIDs(t *testing.T) {
	t.Parallel()

	for _, htmx := range []bool{false, true} {
		content := renderBookViews(bookViewsData{AppName: "library", HTMX: htmx, RuntimeGenerics: true})
		if strings.Contains(content, "Book.objects.get(") {
			t.Fatalf("a stale id would raise DoesNotExist rather than 404 when htmx=%v, got:\n%s", htmx, content)
		}
		for _, want := range []string{"UpdateView", "DeleteView"} {
			if !strings.Contains(content, want) {
				t.Fatalf("edit and delete should inherit the 404 that %s gives them when htmx=%v, got:\n%s", want, htmx, content)
			}
		}
	}
}
