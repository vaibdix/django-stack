package scaffold

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vaibdix/django-cli/internal/config"
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

			e := &Env{Cfg: projectForStack(stack, "catalog", extra...), Reporter: NopReporter{}}
			urls := e.buildProjectURLsContent()

			if !strings.Contains(urls, "path(''") {
				t.Errorf("%s (rest=%v) generates no route for the root URL:\n%s", stack, rest, urls)
			}
		}
	}
}

func TestAPIOnlySendsTheRootURLSomewhereUseful(t *testing.T) {
	t.Parallel()

	withAPI := &Env{Cfg: projectForStack(config.StackAPIOnly, "catalog", config.FeatureRestFramework)}
	urls := withAPI.buildProjectURLsContent()
	if !strings.Contains(urls, "RedirectView.as_view(url='/api/schema/swagger-ui/'") {
		t.Errorf("an API-only project with REST should land on the schema UI:\n%s", urls)
	}
	if !strings.Contains(urls, "from django.views.generic import RedirectView") {
		t.Errorf("RedirectView is used but never imported:\n%s", urls)
	}

	withoutAPI := &Env{Cfg: projectForStack(config.StackAPIOnly, "")}
	urls = withoutAPI.buildProjectURLsContent()
	if !strings.Contains(urls, "RedirectView.as_view(url='/admin/'") {
		t.Errorf("an API-only project without REST should land on the admin:\n%s", urls)
	}
}

func TestAPIOnlyDoesNotMountTheHTMLCrudURLs(t *testing.T) {
	t.Parallel()

	e := &Env{Cfg: projectForStack(config.StackAPIOnly, "catalog", config.FeatureRestFramework)}
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
		Cfg:      projectForStack(config.StackAPIOnly, "catalog", config.FeatureRestFramework),
		Reporter: NopReporter{},
	}
	if err := e.createExampleAPI(projectPath); err != nil {
		t.Fatalf("createExampleAPI: %v", err)
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
			Cfg:      projectForStack(stack, "catalog", config.FeatureRestFramework),
			Reporter: NopReporter{},
		}
		if err := e.createExampleAPI(projectPath); err != nil {
			t.Fatalf("%s: createExampleAPI: %v", stack, err)
		}

		listing := filepath.Join(appDir, "templates", "catalog", "book_list.html")
		data, err := os.ReadFile(listing)
		if err != nil {
			t.Fatalf("%s: expected CRUD templates: %v", stack, err)
		}
		if !strings.Contains(string(data), "{% extends 'base.html' %}") {
			t.Errorf("%s: book_list.html no longer extends base.html", stack)
		}
	}
}

func TestGeneratedTestsOnlyReverseURLsTheStackActuallyRoutes(t *testing.T) {
	t.Parallel()

	apiOnly := &Env{Cfg: projectForStack(config.StackAPIOnly, "catalog", config.FeatureRestFramework)}
	tests := apiOnly.testsPyContent()
	if strings.Contains(tests, "catalog:book_list") {
		t.Errorf("an API-only project generates tests that reverse an HTML route it never mounts, "+
			"so `manage.py test` fails with NoReverseMatch:\n%s", tests)
	}
	if !strings.Contains(tests, "reverse('book-list')") {
		t.Errorf("API-only tests should exercise the router endpoints:\n%s", tests)
	}

	templated := &Env{Cfg: projectForStack(config.StackStandard, "catalog", config.FeatureRestFramework)}
	tests = templated.testsPyContent()
	if !strings.Contains(tests, "catalog:book_list") {
		t.Errorf("a templated project should still test its HTML CRUD pages:\n%s", tests)
	}
}
