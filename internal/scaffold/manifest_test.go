package scaffold

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vaibdix/django-cli/internal/config"
	"github.com/vaibdix/django-cli/internal/scaffold/command/commandtest"
	"github.com/vaibdix/django-cli/internal/scaffold/pyenv"
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
		if got := distributionName(spec); got != want {
			t.Errorf("distributionName(%q) = %q, want %q", spec, got, want)
		}
	}
}

func TestDependencySetDeduplicatesByDistributionNotBySpelling(t *testing.T) {
	t.Parallel()

	var deps dependencySet
	deps.addRuntime("psycopg[binary]")
	deps.addRuntime("psycopg")
	deps.addRuntime("django_environ")
	deps.addRuntime("django-environ")

	if len(deps.runtime) != 2 {
		t.Fatalf("expected 2 distinct distributions, got %d: %v", len(deps.runtime), deps.runtime)
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

func TestProjectDependenciesLandInsideTheProjectTable(t *testing.T) {
	t.Parallel()

	pe := NewPyprojectEditor(t.TempDir(), "demo", PythonToolingMetadata{RequiresPython: ">=3.12"})
	pe.AppendSection("[tool.ruff]\nline-length = 120\n")
	pe.SetProjectDependencies([]string{"django==6.1", "django-environ>=0.14.0"})

	content := pe.content
	projectIdx := strings.Index(content, "[project]")
	depsIdx := strings.Index(content, "dependencies = [")
	ruffIdx := strings.Index(content, "[tool.ruff]")

	if depsIdx < projectIdx || depsIdx > ruffIdx {
		t.Fatalf("dependencies must sit inside [project], before any tool table:\n%s", content)
	}
	if !strings.Contains(content, `    "django==6.1",`) {
		t.Errorf("dependency array is not valid TOML:\n%s", content)
	}
}

func TestSettingDependenciesTwiceReplacesRatherThanAppends(t *testing.T) {
	t.Parallel()

	pe := NewPyprojectEditor(t.TempDir(), "demo", PythonToolingMetadata{RequiresPython: ">=3.12"})
	pe.SetProjectDependencies([]string{"django==6.0"})
	pe.SetProjectDependencies([]string{"django==6.1"})

	if strings.Count(pe.content, "dependencies = [") != 1 {
		t.Fatalf("re-running the manifest step duplicated the array:\n%s", pe.content)
	}
	if strings.Contains(pe.content, "6.0") {
		t.Fatalf("the stale dependency survived:\n%s", pe.content)
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
		e := &Env{Cfg: config.Project{SelectedPythonVersion: c.python, DjangoVersion: c.django}}
		if got := e.pythonToolingMetadata().RequiresPython; got != c.want {
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

func TestManifestRecordsWhatWasInstalledWithTheVersionsItResolved(t *testing.T) {
	t.Parallel()

	projectPath := t.TempDir()

	fake := commandtest.NewFake()
	name, args := pyenv.FreezeCommand(pyenv.Pip, projectPath)
	fake.Responses[commandtest.Call{Name: name, Args: args}.String()] = []byte(
		"asgiref==3.9.1\nDjango==6.1\ndjango-environ==0.14.0\ndjangorestframework==3.18.0\nmypy==2.4.0\nruff==0.16.2\n")

	e := &Env{
		Cfg:      config.Project{ProjectName: "demo", DjangoVersion: "6.1", SelectedPythonVersion: "Python 3.13.2", SetupRuff: true},
		Runner:   fake,
		Reporter: NopReporter{},
	}
	e.deps.addRuntime("django==6.1", PkgEnviron, PkgRestFramework)
	e.deps.addDev(PkgMypy)

	if err := e.writeDependencyManifest(projectPath); err != nil {
		t.Fatalf("writing the manifest: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(projectPath, PyprojectFile))
	if err != nil {
		t.Fatalf("reading %s: %v", PyprojectFile, err)
	}
	manifest := string(content)

	for _, want := range []string{
		`"django==6.1"`,
		`"django-environ>=0.14.0,<0.15"`,
		`"djangorestframework>=3.18.0,<4"`,
		"[dependency-groups]",
		`"ruff>=0.16.2,<0.17"`,
		`"mypy>=2.4.0,<3"`,
	} {
		if !strings.Contains(manifest, want) {
			t.Errorf("manifest is missing %s:\n%s", want, manifest)
		}
	}

	if strings.Contains(manifest, "asgiref") {
		t.Errorf("a transitive dependency was declared as a direct one; that is the lockfile's job:\n%s", manifest)
	}

	version, err := os.ReadFile(filepath.Join(projectPath, PythonVersionFile))
	if err != nil {
		t.Fatalf("reading %s: %v", PythonVersionFile, err)
	}
	if strings.TrimSpace(string(version)) != "3.13.2" {
		t.Errorf("%s = %q, want the interpreter that was actually selected", PythonVersionFile, version)
	}
}

func TestAFailedFreezeFailsTheRunInsteadOfLeavingDependenciesUnpinned(t *testing.T) {
	t.Parallel()

	projectPath := t.TempDir()

	fake := commandtest.NewFake()
	name, args := pyenv.FreezeCommand(pyenv.Pip, projectPath)
	fake.FailWith(commandtest.Call{Name: name, Args: args}.String(), "no such file or directory")

	e := &Env{
		Cfg:      config.Project{ProjectName: "demo", DjangoVersion: "6.1", SelectedPythonVersion: "Python 3.13.2"},
		Runner:   fake,
		Reporter: NopReporter{},
	}
	e.deps.addRuntime("django==6.1", PkgEnviron)

	err := e.writeDependencyManifest(projectPath)
	if err == nil || !strings.Contains(err.Error(), "installed package versions") {
		t.Fatalf("writeDependencyManifest = %v, want the failed freeze reported", err)
	}
	if _, statErr := os.Stat(filepath.Join(projectPath, PyprojectFile)); !os.IsNotExist(statErr) {
		t.Errorf("%s was written with unpinned dependencies after the freeze failed", PyprojectFile)
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
		e := &Env{Cfg: config.Project{
			ProjectName:   "my-shop",
			DatabaseType:  config.SQLite,
			DjangoVersion: version,
		}}

		if err := e.configureSettings(path); err != nil {
			t.Fatalf("configureSettings for Django %s: %v", version, err)
		}

		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("reading settings: %v", err)
		}

		for _, forbidden := range []string{"django.utils.csp", "SECURE_CSP", MiddlewareCSP} {
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
	e := &Env{Cfg: config.Project{
		ProjectName:   "my-shop",
		DatabaseType:  config.SQLite,
		DjangoVersion: "6.1",
	}}

	if err := e.configureSettings(path); err != nil {
		t.Fatalf("configureSettings: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading settings: %v", err)
	}

	for _, want := range []string{"from django.utils.csp import CSP", "SECURE_CSP_REPORT_ONLY", MiddlewareCSP} {
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
		data := multiEnvSettingsData{
			AllAuth: true,
			Mailers: config.DjangoSupportsMailers(version),
			Env:     (&Env{}).productionEnvExpressions(),
		}
		for _, name := range []string{"local.py.tmpl", "production.py.tmpl", "test.py.tmpl"} {
			rendered := render(name, data)
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

func TestPostgresDriverIsNotInstalledForOtherDatabases(t *testing.T) {
	t.Parallel()

	for _, db := range []config.Database{config.SQLite, config.MySQLDocker} {
		if db.UsesPostgres() {
			t.Errorf("%s would pull in the PostgreSQL driver", db)
		}
	}
	for _, db := range []config.Database{config.PostgresDocker} {
		if !db.UsesPostgres() {
			t.Errorf("%s needs the PostgreSQL driver but would not get it", db)
		}
	}
}

func TestOnlyPickedToolsAreDeclaredAsDevDependencies(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		cfg        config.Project
		want, skip []string
	}{
		{cfg: config.Project{}, skip: []string{PkgRuff, PkgDjLint}},
		{cfg: config.Project{SetupRuff: true}, want: []string{PkgRuff}, skip: []string{PkgDjLint}},
		{cfg: config.Project{CreateTemplates: true}, want: []string{PkgDjLint}, skip: []string{PkgRuff}},
	} {
		declared := strings.Join((&Env{Cfg: tc.cfg}).declaredFormattingPackages(), ",")
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
