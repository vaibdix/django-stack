package scaffold

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vaibdix/django-stack/internal/config"
	"github.com/vaibdix/django-stack/internal/report"
)

func projectForStack(stack config.Stack, appName string, extra ...config.Feature) config.Project {
	cfg := config.Project{ProjectName: "my-shop", AppName: appName, DatabaseType: config.SQLite}
	cfg.ApplyFeatures(append(stack.Features(), extra...))
	return cfg
}

func TestEveryStackRoutesTheRootURL(t *testing.T) {
	t.Parallel()

	for _, stack := range config.AllStacks() {
		for _, rest := range []bool{false, true} {
			extra := []config.Feature{}
			if rest {
				extra = append(extra, config.FeatureRestFramework)
			}

			e := &Env{cfg: projectForStack(stack, "catalog", extra...), reporter: report.Nop{}}
			urls := e.buildProjectURLsContent()

			if !strings.Contains(urls, "path(''") {
				t.Errorf("%s (rest=%v) generates no route for the root URL:\n%s", stack, rest, urls)
			}
		}
	}
}

func TestAPIOnlySendsTheRootURLSomewhereUseful(t *testing.T) {
	t.Parallel()

	withAPI := &Env{cfg: projectForStack(config.StackAPIOnly, "catalog", config.FeatureRestFramework)}
	urls := withAPI.buildProjectURLsContent()
	if !strings.Contains(urls, "RedirectView.as_view(url='/api/schema/swagger-ui/'") {
		t.Errorf("an API-only project with REST should land on the schema UI:\n%s", urls)
	}
	if !strings.Contains(urls, "from django.views.generic import RedirectView") {
		t.Errorf("RedirectView is used but never imported:\n%s", urls)
	}

	withoutAPI := &Env{cfg: projectForStack(config.StackAPIOnly, "")}
	urls = withoutAPI.buildProjectURLsContent()
	if !strings.Contains(urls, "RedirectView.as_view(url='/admin/'") {
		t.Errorf("an API-only project without REST should land on the admin:\n%s", urls)
	}
}

func TestAPIOnlyDoesNotMountTheHTMLCrudURLs(t *testing.T) {
	t.Parallel()

	e := &Env{cfg: projectForStack(config.StackAPIOnly, "catalog", config.FeatureRestFramework)}
	urls := e.buildProjectURLsContent()

	if !strings.Contains(urls, "include('catalog.api')") {
		t.Errorf("the API routes are missing:\n%s", urls)
	}
	if strings.Contains(urls, "include('catalog.urls')") {
		t.Errorf("an API-only project mounts catalog.urls, but no urls.py is generated for it:\n%s", urls)
	}
}

func TestAPIOnlyGeneratesNoHTMLFacingFiles(t *testing.T) {
	t.Parallel()

	projectPath := t.TempDir()
	appDir := filepath.Join(projectPath, "catalog")
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		t.Fatalf("creating the app directory: %v", err)
	}

	e := &Env{
		cfg:      projectForStack(config.StackAPIOnly, "catalog", config.FeatureRestFramework),
		reporter: report.Nop{},
	}
	if err := e.writeExampleApp(projectPath, generateAPIPackage(e.cfg.AppName)); err != nil {
		t.Fatalf("writeExampleApp: %v", err)
	}

	for _, name := range []string{"models.py", "admin.py", "api/__init__.py", "api/serializers.py", "api/views.py", "api/urls.py"} {
		if _, err := os.Stat(filepath.Join(appDir, filepath.FromSlash(name))); err != nil {
			t.Errorf("an API-only project still needs %s: %v", name, err)
		}
	}
	for _, name := range []string{"forms.py", "views.py", "urls.py", "templates"} {
		if _, err := os.Stat(filepath.Join(appDir, name)); err == nil {
			t.Errorf("%s was generated for an API-only project; nothing routes to it", name)
		}
	}

	api, err := os.ReadFile(filepath.Join(appDir, "api", "views.py"))
	if err != nil {
		t.Fatalf("reading api/views.py: %v", err)
	}
	if strings.Contains(string(api), "from ..forms") || strings.Contains(string(api), "from ..views") {
		t.Errorf("api/views.py imports a module that an API-only project does not generate:\n%s", api)
	}
}

func TestTemplatedStacksStillGenerateTheirCrudPages(t *testing.T) {
	t.Parallel()

	for _, stack := range []config.Stack{
		config.StackStandard, config.StackHTMX, config.StackTailwind, config.StackTailwindHTMX,
	} {
		projectPath := t.TempDir()
		appDir := filepath.Join(projectPath, "catalog")
		if err := os.MkdirAll(appDir, 0o755); err != nil {
			t.Fatalf("creating the app directory: %v", err)
		}

		e := &Env{
			cfg:      projectForStack(stack, "catalog", config.FeatureRestFramework),
			reporter: report.Nop{},
		}
		if err := e.writeExampleApp(projectPath, generateAPIPackage(e.cfg.AppName)); err != nil {
			t.Fatalf("%s: writeExampleApp: %v", stack, err)
		}

		listing := filepath.Join(appDir, "templates", "catalog", "book_list.html")
		data, err := os.ReadFile(listing)
		if err != nil {
			t.Fatalf("%s: expected CRUD templates: %v", stack, err)
		}
		if !strings.Contains(string(data), "{% extends \"base.html\" %}") {
			t.Errorf("%s: book_list.html no longer extends base.html", stack)
		}
	}
}

func TestGeneratedTestsOnlyReverseURLsTheStackActuallyRoutes(t *testing.T) {
	t.Parallel()

	apiOnly := &Env{cfg: projectForStack(config.StackAPIOnly, "catalog", config.FeatureRestFramework)}
	tests := apiOnly.testFiles()[filepath.Join("tests", "test_api.py")]
	if strings.Contains(tests, "catalog:book_list") {
		t.Errorf("an API-only project generates tests that reverse an HTML route it never mounts, "+
			"so `manage.py test` fails with NoReverseMatch:\n%s", tests)
	}
	if !strings.Contains(tests, "reverse('book-list')") {
		t.Errorf("API-only tests should exercise the router endpoints:\n%s", tests)
	}

	templated := &Env{cfg: projectForStack(config.StackStandard, "catalog", config.FeatureRestFramework)}
	tests = templated.testFiles()[filepath.Join("tests", "test_pages.py")]
	if !strings.Contains(tests, "catalog:book_list") {
		t.Errorf("a templated project should still test its HTML CRUD pages:\n%s", tests)
	}
}

func TestTheHTMXStackGetsTheHTMXCrudWithoutAnAPI(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "catalog"), 0o755); err != nil {
		t.Fatalf("creating the app: %v", err)
	}
	cfg := config.Project{ProjectName: "demo", AppName: "catalog", DjangoVersion: "6.1"}
	cfg.ApplyFeatures(config.StackHTMX.Features())
	e := &Env{cfg: cfg, reporter: report.Nop{}}

	if err := e.writeBookApp(root); err != nil {
		t.Fatalf("writeBookApp: %v", err)
	}

	views, err := os.ReadFile(filepath.Join(root, "catalog", "views.py"))
	if err != nil {
		t.Fatalf("an HTMX project without an API has no views: %v", err)
	}
	if !strings.Contains(string(views), "HtmxTemplateMixin") {
		t.Errorf("the HTMX stack's views do not use HTMX:\n%s", views)
	}
	for _, name := range []string{"partials/book_list_items.html", "partials/book_form_partial.html"} {
		if _, err := os.Stat(filepath.Join(root, "catalog", "templates", "catalog", name)); err != nil {
			t.Errorf("the HTMX partial %s is missing: %v", name, err)
		}
	}
	if _, err := os.Stat(filepath.Join(root, "catalog", "api")); !os.IsNotExist(err) {
		t.Error("an api/ package was written although no API add-on was picked")
	}
}

func TestAPIOnlyErrorHandlersRenderNoTemplates(t *testing.T) {
	t.Parallel()

	for _, api := range []config.Feature{config.FeatureRestFramework, config.FeatureNinja} {
		e := &Env{cfg: projectForStack(config.StackAPIOnly, "catalog", api)}
		views := e.buildProjectViewsContent()

		if strings.Contains(views, "render(") || strings.Contains(views, ".html") {
			t.Errorf("%s: an API-only project has no templates, so a handler that renders one turns every 404 into a crash:\n%s", api, views)
		}
		for _, handler := range []string{"bad_request", "permission_denied", "page_not_found", "server_error"} {
			if !strings.Contains(views, "def "+handler+"(") {
				t.Errorf("%s: urls.py points handler at %s, but views.py does not define it:\n%s", api, handler, views)
			}
		}
		if !strings.Contains(views, "JsonResponse({'detail': 'Not found.'}, status=404)") {
			t.Errorf("%s: the 404 handler should answer in the API's own JSON error shape:\n%s", api, views)
		}
	}
}

func TestPageStacksKeepDjangosErrorViews(t *testing.T) {
	t.Parallel()

	for _, stack := range []config.Stack{config.StackStandard, config.StackHTMX, config.StackTailwind, config.StackTailwindHTMX} {
		e := &Env{cfg: projectForStack(stack, "catalog")}
		if urls := e.buildProjectURLsContent(); strings.Contains(urls, "handler") {
			t.Errorf("%s: Django's own error views already render templates/404.html and render 500.html "+
				"with an empty Context; a custom handler adds nothing and risks a second failure:\n%s", stack, urls)
		}
		if views := e.buildProjectViewsContent(); strings.Contains(views, "server_error") || strings.Contains(views, "render(") {
			t.Errorf("%s: views.py still defines error handlers:\n%s", stack, views)
		}
	}
}

func TestProjectsWithPagesAndAnAPITestBoth(t *testing.T) {
	t.Parallel()

	for _, api := range []config.Feature{config.FeatureRestFramework, config.FeatureNinja} {
		e := &Env{cfg: projectForStack(config.StackHTMX, "catalog", api)}
		files := e.testFiles()

		pages, api := files[filepath.Join("tests", "test_pages.py")], files[filepath.Join("tests", "test_api.py")]
		if !strings.Contains(pages, "catalog:book_create") {
			t.Errorf("the pages are untested:\n%v", files)
		}
		if !strings.Contains(api, "APITests(TestCase)") {
			t.Errorf("the API is untested when the project also has pages:\n%v", files)
		}
		if _, ok := files[filepath.Join("tests", "__init__.py")]; !ok {
			t.Errorf("tests/ is not a package, so the test runner never finds it:\n%v", files)
		}
	}
}

func TestModelTestsAreWrittenOnce(t *testing.T) {
	t.Parallel()

	for _, cfg := range []config.Project{
		projectForStack(config.StackStandard, "catalog"),
		projectForStack(config.StackHTMX, "catalog", config.FeatureRestFramework),
		projectForStack(config.StackAPIOnly, "catalog", config.FeatureNinja),
	} {
		files := (&Env{cfg: cfg}).testFiles()
		var holders []string
		for name, content := range files {
			if strings.Contains(content, "ModelTests(TestCase)") {
				holders = append(holders, name)
			}
		}
		if len(holders) != 1 || holders[0] != filepath.Join("tests", "test_models.py") {
			t.Errorf("model tests should live only in tests/test_models.py, found them in %v", holders)
		}
	}
}

func TestPagesWithoutAllauthStillLetOrdinaryUsersSignIn(t *testing.T) {
	t.Parallel()

	e := &Env{cfg: projectForStack(config.StackStandard, "catalog")}
	if urls := e.buildProjectURLsContent(); !strings.Contains(urls, "include('django.contrib.auth.urls')") {
		t.Errorf("without allauth the only sign-in page is the admin's, which refuses non-staff users:\n%s", urls)
	}

	withAllauth := &Env{cfg: projectForStack(config.StackStandard, "catalog", config.FeatureAllAuth)}
	if urls := withAllauth.buildProjectURLsContent(); strings.Contains(urls, "django.contrib.auth.urls") {
		t.Errorf("allauth already owns /accounts/; mounting django.contrib.auth there too shadows it:\n%s", urls)
	}

	apiOnly := &Env{cfg: projectForStack(config.StackAPIOnly, "catalog", config.FeatureRestFramework)}
	if urls := apiOnly.buildProjectURLsContent(); strings.Contains(urls, "django.contrib.auth.urls") {
		t.Errorf("an API-only project has no templates for the auth views:\n%s", urls)
	}
}
