package scaffold

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/vaibdix/django-cli/internal/config"
)

func ninjaProject(appName string, extra ...config.Feature) config.Project {
	cfg := config.Project{
		ProjectName:   "my-shop",
		AppName:       appName,
		DjangoVersion: "6.0",
		DatabaseType:  config.SQLite,
	}
	cfg.ApplyFeatures(append([]config.Feature{config.FeatureNinja}, extra...))
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

	e := &Env{Cfg: ninjaProject("catalog"), Reporter: NopReporter{}}
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

	e := &Env{Cfg: ninjaProject("catalog"), Reporter: NopReporter{}}
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

func TestTheNinjaSettingsBlockUsesTheNamesNinjaReads(t *testing.T) {
	t.Parallel()

	for _, want := range []string{
		"NINJA_PAGINATION_CLASS",
		"NINJA_PAGINATION_PER_PAGE",
		"NINJA_DEFAULT_THROTTLE_RATES",
	} {
		if !strings.Contains(ninjaSettingsBlock, want) {
			t.Errorf("the settings block is missing %q, so ninja falls back to its own default", want)
		}
	}
}

func TestTheNinjaEndpointsDocumentTheDocsURL(t *testing.T) {
	t.Parallel()

	e := &Env{Cfg: ninjaProject("catalog"), Reporter: NopReporter{}}
	md := e.generateNinjaEndpointsMd()

	if !strings.Contains(md, "/api/v1/docs") {
		t.Errorf("API_ENDPOINTS.md never names the docs URL:\n%s", md)
	}
	if strings.Contains(md, "swagger-ui") || strings.Contains(md, "api-auth") {
		t.Errorf("API_ENDPOINTS.md still documents the DRF routes:\n%s", md)
	}
}

func TestAnAPIOnlyNinjaProjectSendsTheRootURLToTheDocs(t *testing.T) {
	t.Parallel()

	cfg := ninjaProject("catalog")
	e := &Env{Cfg: cfg, Reporter: NopReporter{}}

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

	e := &Env{Cfg: ninjaProject("catalog"), Reporter: NopReporter{}}
	if err := e.createExampleAPI(root, generateNinjaPackage("catalog", "my-shop")); err != nil {
		t.Fatalf("createExampleAPI: %v", err)
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

func TestNinjaIsOfferedOnlyWhereItInstalls(t *testing.T) {
	t.Parallel()

	unsupported := []string{"6.1", "6.1.2", "7.0"}
	supported := []string{"4.2", "5.0", "5.2.12", "6.0", "6.0.3"}

	for _, v := range unsupported {
		if config.DjangoSupportsNinja(v) {
			t.Errorf("django %s is offered django-ninja, but every release caps at Django<6.1", v)
		}
	}
	for _, v := range supported {
		if !config.DjangoSupportsNinja(v) {
			t.Errorf("django %s supports django-ninja but the option is withheld", v)
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

	e := &Env{Cfg: config.Project{ProjectName: "demo", AppName: "catalog", SetupNinja: true}}
	tests := e.testsPyContent()
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
