package scaffold

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/vaibdix/django-stack/internal/config"
	"github.com/vaibdix/django-stack/internal/report"
)

func ninjaProject(appName string) config.Choices {
	cfg := config.Choices{
		Name:     "my-shop",
		App:      appName,
		Django:   "6.0",
		Database: config.SQLite,
	}
	cfg.Stack, cfg.AddOns = config.StackAPIOnly, config.NewSet(config.Ninja)
	return cfg
}

func ninjaFiles(t *testing.T) map[string]string {
	t.Helper()
	return generateNinjaPackage("catalog", "my-shop")
}

func TestTheNinjaPackageShipsTheFourFilesTheProjectImports(t *testing.T) {
	t.Parallel()

	files := ninjaFiles(t)

	for _, name := range []string{"__init__.py", "api.py", "router.py", "schemas.py"} {
		if _, ok := files[name]; !ok {
			t.Errorf("api/%s is missing from the ninja package", name)
		}
	}
}

func TestTheNinjaAPIIsReachableFromTheProjectURLs(t *testing.T) {
	t.Parallel()

	e := &Env{cfg: ninjaProject("catalog"), reporter: report.Nop{}}
	urls := e.buildProjectURLsContent()

	if !strings.Contains(urls, "from catalog.api import api") {
		t.Errorf("urls.py never imports the ninja api instance:\n%s", urls)
	}
	if !strings.Contains(urls, "path('api/v1/', api.urls),") {
		t.Errorf("urls.py never mounts the ninja api:\n%s", urls)
	}
	if strings.Contains(urls, "drf_spectacular") {
		t.Errorf("a ninja project imports drf-spectacular views:\n%s", urls)
	}
	if strings.Contains(urls, "rest_framework") {
		t.Errorf("a ninja project routes to rest_framework:\n%s", urls)
	}
}

func TestTheNinjaDocsLinkIsReversedNotHardcoded(t *testing.T) {
	t.Parallel()

	e := &Env{cfg: ninjaProject("catalog"), reporter: report.Nop{}}
	if got := e.apiDocsRoute(); got != "api:openapi-view" {
		t.Errorf("apiDocsRoute() = %q; ninja names its docs view openapi-view "+
			"inside the urls_namespace the api instance declares", got)
	}

	api := generateNinjaPackage("catalog", "my-shop")["api.py"]
	if !strings.Contains(api, "urls_namespace='api'") {
		t.Errorf("the api instance does not pin its namespace, so {%% url 'api:openapi-view' %%} "+
			"cannot resolve:\n%s", api)
	}
}

func TestNinjaWritesAreAuthenticatedAndReadsAreNot(t *testing.T) {
	t.Parallel()

	router := ninjaFiles(t)["router.py"]

	for _, write := range []string{"@router.post(", "@router.put(", "@router.delete("} {
		block := operationBlock(router, write)
		if !strings.Contains(block, "auth=django_auth") {
			t.Errorf("%s is writable by anonymous callers:\n%s", write, block)
		}
	}

	for _, read := range []string{"@router.get('', ", "@router.get('/recent'"} {
		block := operationBlock(router, read)
		if strings.Contains(block, "auth=django_auth") {
			t.Errorf("%s requires a login, so the demo API cannot be browsed:\n%s", read, block)
		}
	}
}

func operationBlock(source, marker string) string {
	start := strings.Index(source, marker)
	if start < 0 {
		return ""
	}
	rest := source[start:]
	if end := strings.Index(rest, "\ndef "); end > 0 {
		return rest[:end]
	}
	return rest
}

func TestNinjaListsArePaginated(t *testing.T) {
	t.Parallel()

	router := ninjaFiles(t)["router.py"]

	if strings.Count(router, "@paginate(PageNumberPagination") != 2 {
		t.Errorf("the list and recent endpoints are not both paginated, so a large table "+
			"is returned in one response:\n%s", router)
	}
}

func TestNinjaQueriesGoThroughTheModelManager(t *testing.T) {
	t.Parallel()

	router := ninjaFiles(t)["router.py"]

	for _, want := range []string{"Book.objects.search(", ".recent()"} {
		if !strings.Contains(router, want) {
			t.Errorf("the router re-implements %q instead of calling the model queries:\n%s", want, router)
		}
	}
}

func TestNinjaIsNeverAddedToInstalledApps(t *testing.T) {
	t.Parallel()

	files := ninjaFiles(t)
	for name, content := range files {
		if strings.Contains(content, "INSTALLED_APPS") {
			t.Errorf("api/%s touches INSTALLED_APPS; django-ninja is not a Django app:\n%s", name, content)
		}
	}
}

func TestNinjaPaginationAndThrottlesAreDeclaredOnce(t *testing.T) {
	t.Parallel()

	files := ninjaFiles(t)
	if !strings.Contains(files["router.py"], "@paginate(PageNumberPagination, page_size=PAGE_SIZE)") {
		t.Errorf("the router no longer states its pagination:\n%s", files["router.py"])
	}
	if !strings.Contains(files["api.py"], "AnonRateThrottle('100/h')") {
		t.Errorf("the API no longer states its throttles:\n%s", files["api.py"])
	}
}

func TestAnAPIOnlyNinjaProjectSendsTheRootURLToTheDocs(t *testing.T) {
	t.Parallel()

	cfg := ninjaProject("catalog")
	e := &Env{cfg: cfg, reporter: report.Nop{}}

	if !cfg.APIOnly() {
		t.Fatal("a bare ninja add-on should leave the project API-only")
	}
	if got := e.apiOnlyHomeTarget(); got != "/api/v1/docs" {
		t.Errorf("apiOnlyHomeTarget() = %q, want the ninja docs page", got)
	}
}

func TestTheNinjaExampleWritesItsPackageIntoTheApp(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	appDir := filepath.Join(root, "catalog")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	e := &Env{cfg: ninjaProject("catalog"), reporter: report.Nop{}}
	if err := e.writeExampleApp(root, generateNinjaPackage("catalog", "my-shop")); err != nil {
		t.Fatalf("writeExampleApp: %v", err)
	}

	for _, name := range []string{
		"models.py", "admin.py",
		filepath.Join("api", "__init__.py"), filepath.Join("api", "api.py"),
		filepath.Join("api", "router.py"), filepath.Join("api", "schemas.py"),
		filepath.Join("management", "commands", "create_sample_data.py"),
	} {
		if _, err := os.Stat(filepath.Join(appDir, name)); err != nil {
			t.Errorf("a ninja project still needs %s: %v", name, err)
		}
	}
}

func TestTheGeneratedNinjaPythonSurvivesRuff(t *testing.T) {
	t.Parallel()

	cmd, ok := ruffCommand(t, "--version")
	if !ok {
		t.Skip("neither ruff nor uvx is installed")
	}
	if err := cmd.Run(); err != nil {
		t.Skipf("ruff is not runnable: %v", err)
	}

	const app = "catalog"

	root := t.TempDir()
	writeGeneratedRuffConfig(t, root, app)

	apiDir := filepath.Join(root, app, "api")
	if err := os.MkdirAll(apiDir, 0o755); err != nil {
		t.Fatalf("creating %s: %v", apiDir, err)
	}

	for name, content := range generateNinjaPackage(app, "lintcheck") {
		if err := os.WriteFile(filepath.Join(apiDir, name), []byte(content), 0o644); err != nil {
			t.Fatalf("writing %s: %v", name, err)
		}
	}

	check, _ := ruffCommand(t, "check", "--no-cache", "--no-fix", ".")
	check.Dir = root
	out, err := check.CombinedOutput()
	if err != nil {
		t.Errorf("ruff rejects the generated ninja package, which aborts the scaffold:\n%s", out)
	}
}

var reversedNinjaRoute = regexp.MustCompile(`reverse\('api:(\w+)'`)

func TestNinjaProjectsGetTestsThatReverseTheirOwnRoutes(t *testing.T) {
	t.Parallel()

	e := &Env{cfg: config.Choices{Name: "demo", App: "catalog", AddOns: config.NewSet(config.Ninja)}}
	tests := e.testFiles()[filepath.Join("tests", "test_api.py")]
	if strings.Contains(tests, "book-list") {
		t.Fatalf("an API-only Ninja project was given the DRF test file:\n%s", tests)
	}

	router := generateNinjaPackage("catalog", "demo")["router.py"]
	names := reversedNinjaRoute.FindAllStringSubmatch(tests, -1)
	if len(names) == 0 {
		t.Fatalf("the Ninja tests reverse no API route, so they exercise nothing:\n%s", tests)
	}
	for _, name := range names {
		if !strings.Contains(router, "def "+name[1]+"(") {
			t.Errorf("tests reverse api:%s, but router.py defines no such operation", name[1])
		}
	}
}

func TestNinjaEnablesThePydanticMypyPlugin(t *testing.T) {
	t.Parallel()

	written := mypyConfigFor(t, ninjaProject("catalog"))
	if !strings.Contains(written, `"pydantic.mypy"`) {
		t.Errorf("Ninja schemas are pydantic models; without the plugin, disallow_any_explicit "+
			"rejects every Schema subclass:\n%s", written)
	}
}

func TestAPIDocsServeTheirAssetsFromTheProject(t *testing.T) {
	t.Parallel()

	drf := &Env{cfg: projectForStack(config.StackAPIOnly, "catalog", config.RestFramework)}
	urls := drf.buildProjectURLsContent()
	if !strings.Contains(urls, "SpectacularSwaggerSplitView") {
		t.Errorf("SpectacularSwaggerView renders an inline script that script-src 'self' blocks:\n%s", urls)
	}
	if strings.Contains(strings.ToLower(urls), "redoc") {
		t.Errorf("ReDoc needs an inline script and duplicates the Swagger UI:\n%s", urls)
	}

	plan := drf.packagePlan()
	if !slices.Contains(plan.runtime, pkgDRFSpectacularSidecar) {
		t.Errorf("without the sidecar, Swagger UI loads from a CDN the CSP does not allow: %v", plan.runtime)
	}
}
