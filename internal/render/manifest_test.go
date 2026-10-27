package render

import (
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/vaibdix/django-stack/internal/config"
)

func TestDistributionNameStripsExtrasAndSpecifiers(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"psycopg[binary]":       "psycopg",
		"psycopg[binary]>=3.2":  "psycopg",
		"django==6.1":           "django",
		"django-environ>=0.14":  "django-environ",
		"djangorestframework":   "djangorestframework",
		"Django_Stubs":          "django-stubs",
		"drf-spectacular; x":    "drf-spectacular",
		"  django-htmx  >= 1.0": "django-htmx",
	}

	for spec, want := range cases {
		if got := DistributionName(spec); got != want {
			t.Errorf("distributionName(%q) = %q, want %q", spec, got, want)
		}
	}
}

func TestDependencySetDeduplicatesByDistributionNotBySpelling(t *testing.T) {
	t.Parallel()

	var deps DependencySet
	deps.AddRuntime("psycopg[binary]")
	deps.AddRuntime("psycopg")
	deps.AddRuntime("django_environ")
	deps.AddRuntime("django-environ")

	if len(deps.Runtime) != 2 {
		t.Fatalf("expected 2 distinct distributions, got %d: %v", len(deps.Runtime), deps.Runtime)
	}
}

func TestPinSpecUsesTheVersionThatWasActuallyInstalled(t *testing.T) {
	t.Parallel()

	versions := map[string]string{"django-environ": "0.14.0", "psycopg": "3.3.4"}

	for spec, want := range map[string]string{
		"django-environ":  "django-environ>=0.14.0,<0.15",
		"psycopg[binary]": "psycopg[binary]>=3.3.4,<4",
		"django==6.1":     "django==6.1",
	} {
		if got, err := pinSpec(spec, versions); err != nil || got != want {
			t.Errorf("pinSpec(%q) = %q, %v; want %q", spec, got, err, want)
		}
	}
	if got, err := pinSpec("unheard-of", versions); err == nil {
		t.Errorf("pinSpec left a package the freeze never saw unpinned (%q) instead of failing", got)
	}
}

func TestEveryFloorCarriesACeilingSoAMajorReleaseCannotBeResolvedInto(t *testing.T) {
	t.Parallel()

	versions := map[string]string{
		"djangorestframework": "3.18.2",
		"pydantic":            "2.13.1",
		"ruff":                "0.16.2",
	}

	for spec, want := range map[string]string{
		"djangorestframework": "djangorestframework>=3.18.2,<4",
		"pydantic":            "pydantic>=2.13.1,<3",
		"ruff":                "ruff>=0.16.2,<0.17",
	} {
		if got, _ := pinSpec(spec, versions); got != want {
			t.Errorf("pinSpec(%q) = %q, want %q; an unbounded floor lets a fresh resolve pull a new major", spec, got, want)
		}
	}

	if got := nextIncompatibleVersion("not-a-version"); got != "" {
		t.Errorf("an unparseable version produced a ceiling anyway: %q", got)
	}
}

func TestRequiresPythonRespectsTheDjangoFloor(t *testing.T) {
	t.Parallel()

	cases := []struct {
		python, django, want string
	}{
		{"Python 3.11.9", "6.1", ">=3.12"},
		{"Python 3.11.9", "5.2.12", ">=3.11"},
		{"Python 3.13.2", "6.1", ">=3.13"},
		{"Python 3.10.4", "6.0.3", ">=3.12"},
		{"Python 3.12.1", "", ">=3.12"},
	}

	for _, c := range cases {
		e := &Project{Cfg: mustSpec(config.Choices{Django: c.django}), Resolved: config.Resolved{PythonVersion: c.python}}
		if got := e.PythonToolingMetadata().RequiresPython; got != c.want {
			t.Errorf("Python %q + Django %q gave requires-python %q, want %q; "+
				"uv refuses to resolve a project whose floor is below its pinned Django",
				c.python, c.django, got, c.want)
		}
	}
}

func TestEveryDjangoSeriesOfferedHasAPythonFloor(t *testing.T) {
	t.Parallel()

	for _, rule := range config.DjangoCompatibilityRules {
		floor, ok := config.PythonFloorForDjango(rule.Series + ".0")
		if !ok {
			t.Errorf("no Python floor resolves for the %s series", rule.Series)
			continue
		}
		if floor != rule.MinMinor {
			t.Errorf("Django %s floor = 3.%d, want 3.%d", rule.Series, floor, rule.MinMinor)
		}
	}
}

func TestNativeCSPIsOnlyEmittedForDjangoThatHasIt(t *testing.T) {
	t.Parallel()

	cases := map[string]bool{
		"6.1": true, "6.0.3": true, "7.0": true,
		"5.2.12": false, "5.1.6": false, "4.2.28": false, "": false,
	}

	for version, want := range cases {
		if got := config.DjangoSupportsNativeCSP(version); got != want {
			t.Errorf("DjangoSupportsNativeCSP(%q) = %v, want %v", version, got, want)
		}
	}
}

func TestSettingsForOlderDjangoDoNotImportTheCSPModule(t *testing.T) {
	t.Parallel()

	for _, version := range []string{"4.2.28", "5.2.12"} {
		path := newSettingsFixture(t)
		e := &Project{Cfg: mustSpec(config.Choices{
			Name:     "my-shop",
			Database: config.SQLite,
			Django:   version,
		})}

		if err := configureSettingsAt(e, path); err != nil {
			t.Fatalf("configureSettings for Django %s: %v", version, err)
		}

		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("reading settings: %v", err)
		}

		for _, forbidden := range []string{"django.utils.csp", "SECURE_CSP", middlewareCSP} {
			if strings.Contains(string(content), forbidden) {
				t.Errorf("Django %s has no native CSP, but the settings reference %s; "+
					"every manage.py command dies with ModuleNotFoundError", version, forbidden)
			}
		}
	}
}

func TestSettingsForDjango6EmitTheCSPPolicyAndItsMiddleware(t *testing.T) {
	t.Parallel()

	path := newSettingsFixture(t)
	e := &Project{Cfg: mustSpec(config.Choices{
		Name:     "my-shop",
		Database: config.SQLite,
		Django:   "6.1",
	})}

	if err := configureSettingsAt(e, path); err != nil {
		t.Fatalf("configureSettings: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading settings: %v", err)
	}

	for _, want := range []string{"from django.utils.csp import CSP", "SECURE_CSP_REPORT_ONLY", middlewareCSP} {
		if !strings.Contains(string(content), want) {
			t.Errorf("Django 6.1 settings are missing %s", want)
		}
	}
}

func TestEmailSettingsMatchWhatTheDjangoVersionAccepts(t *testing.T) {
	t.Parallel()

	cases := map[string]bool{
		"6.1": true, "6.1.2": true, "7.0": true,
		"6.0.3": false, "5.2.12": false, "4.2.28": false, "": false,
	}

	for version, wantMailers := range cases {
		if got := config.DjangoSupportsMailers(version); got != wantMailers {
			t.Errorf("DjangoSupportsMailers(%q) = %v, want %v", version, got, wantMailers)
		}
	}

	for version, wantMailers := range cases {
		data := MultiEnvSettingsData{
			AllAuth: true,
			Mailers: config.DjangoSupportsMailers(version),
			Env:     (&Project{}).ProductionEnvExpressions(),
		}
		for _, name := range []string{"local.py.tmpl", "production.py.tmpl", "test.py.tmpl"} {
			rendered := Render(name, data)
			hasMailers := strings.Contains(rendered, "MAILERS")
			hasLegacy := strings.Contains(rendered, "EMAIL_BACKEND")

			if hasMailers && hasLegacy {
				t.Errorf("%s for Django %q sets both MAILERS and EMAIL_BACKEND; "+
					"Django 6.1 raises ImproperlyConfigured when both are defined", name, version)
			}
			if hasMailers != wantMailers {
				t.Errorf("%s for Django %q: MAILERS present = %v, want %v", name, version, hasMailers, wantMailers)
			}
		}
	}
}

func TestEachDatabaseInstallsOnlyItsOwnDriver(t *testing.T) {
	t.Parallel()

	want := map[config.Database]string{config.SQLite: "", config.PostgresDocker: "psycopg[binary]", config.MySQLDocker: "mysqlclient"}
	for _, db := range config.Databases() {
		e := &Project{Cfg: mustSpec(config.Choices{Django: "6.1.2", Database: db})}
		runtime := e.PackagePlan().Runtime
		for _, driver := range []string{"psycopg[binary]", "mysqlclient"} {
			if slices.Contains(runtime, driver) != (driver == want[db]) {
				t.Errorf("%s: runtime packages %v, want only the %q driver", db, runtime, want[db])
			}
		}
	}
}

func TestOnlyPickedToolsAreDeclaredAsDevDependencies(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		cfg        config.Choices
		want, skip []string
	}{
		{cfg: config.Choices{}, skip: []string{PkgRuff, PkgDjLint}},
		{cfg: config.Choices{AddOns: config.NewSet(config.Ruff)}, want: []string{PkgRuff}, skip: []string{PkgDjLint}},
		{cfg: config.Choices{Stack: config.StackStandard}, want: []string{PkgDjLint}, skip: []string{PkgRuff}},
	} {
		declared := strings.Join((&Project{Cfg: mustSpec(tc.cfg)}).DeclaredFormattingPackages(), ",")
		for _, pkg := range tc.want {
			if !strings.Contains(declared, pkg) {
				t.Errorf("%+v: %s is not declared", tc.cfg, pkg)
			}
		}
		for _, pkg := range tc.skip {
			if strings.Contains(declared, pkg) {
				t.Errorf("%+v: %s is declared although it was not picked", tc.cfg, pkg)
			}
		}
	}
}

func TestProjectDependenciesLandInsideTheProjectTable(t *testing.T) {
	t.Parallel()

	e := &Project{Cfg: mustSpec(config.Choices{Name: "demo", Django: "6.1.2", AddOns: config.NewSet(config.Ruff, config.Mypy)}), Resolved: config.Resolved{PythonVersion: "Python 3.13.2"}}
	content := e.RenderPyproject([]string{"django==6.1", "django-environ>=0.14.0"}, []string{"ruff==0.15.0"})

	projectIdx := strings.Index(content, "[project]")
	depsIdx := strings.Index(content, "dependencies = [")
	ruffIdx := strings.Index(content, "[tool.ruff]")
	groupsIdx := strings.Index(content, "[dependency-groups]")

	if depsIdx < projectIdx || depsIdx > ruffIdx || groupsIdx < ruffIdx {
		t.Fatalf("dependencies must sit inside [project], before any tool table, and the dev group last:\n%s", content)
	}
	if strings.Count(content, "dependencies = [") != 1 || !strings.Contains(content, `    "django==6.1",`) {
		t.Errorf("dependency array is not valid TOML:\n%s", content)
	}
}
