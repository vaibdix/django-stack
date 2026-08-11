package scaffold

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vaibdix/django-cli/internal/config"
	"github.com/vaibdix/django-cli/internal/scaffold/command"
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

	if got := pinSpec("django-environ", versions); got != "django-environ>=0.14.0" {
		t.Errorf("pinSpec added the wrong floor: %q", got)
	}
	if got := pinSpec("psycopg[binary]", versions); got != "psycopg[binary]>=3.3.4" {
		t.Errorf("pinSpec dropped the extra: %q", got)
	}
	if got := pinSpec("django==6.1", versions); got != "django==6.1" {
		t.Errorf("pinSpec overwrote an explicit pin: %q", got)
	}
	if got := pinSpec("unheard-of", versions); got != "unheard-of" {
		t.Errorf("pinSpec invented a floor for a package it never saw: %q", got)
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

	fake := command.NewFake()
	name, args := pyenv.FreezeCommand(projectPath)
	fake.Responses[command.Call{Name: name, Args: args}.String()] = []byte(
		"asgiref==3.9.1\nDjango==6.1\ndjango-environ==0.14.0\ndjangorestframework==3.18.0\nruff==0.16.2\n")

	e := &Env{
		Cfg:      config.Project{ProjectName: "demo", DjangoVersion: "6.1", SelectedPythonVersion: "Python 3.13.2"},
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
		`"django-environ>=0.14.0"`,
		`"djangorestframework>=3.18.0"`,
		"[dependency-groups]",
		`"ruff>=0.16.2"`,
		`"` + PkgMypy + `"`,
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

func TestManifestSurvivesAFreezeThatFails(t *testing.T) {
	t.Parallel()

	projectPath := t.TempDir()

	fake := command.NewFake()
	name, args := pyenv.FreezeCommand(projectPath)
	fake.FailWith(command.Call{Name: name, Args: args}.String(), "no such file or directory")

	e := &Env{
		Cfg:      config.Project{ProjectName: "demo", DjangoVersion: "6.1", SelectedPythonVersion: "Python 3.13.2"},
		Runner:   fake,
		Reporter: NopReporter{},
	}
	e.deps.addRuntime("django==6.1", PkgEnviron)

	if err := e.writeDependencyManifest(projectPath); err != nil {
		t.Fatalf("a failed freeze must not abort the scaffold: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(projectPath, PyprojectFile))
	if err != nil {
		t.Fatalf("reading %s: %v", PyprojectFile, err)
	}
	if !strings.Contains(string(content), `"`+PkgEnviron+`"`) {
		t.Errorf("dependencies must still be declared without floors when the freeze fails:\n%s", content)
	}
}

func TestPostgresDriverIsNotInstalledForOtherDatabases(t *testing.T) {
	t.Parallel()

	for _, db := range []config.Database{config.SQLite, config.MySQLDocker} {
		if db.UsesPostgres() {
			t.Errorf("%s would pull in the PostgreSQL driver", db)
		}
	}
	for _, db := range []config.Database{config.PostgresDocker, config.PostgresString} {
		if !db.UsesPostgres() {
			t.Errorf("%s needs the PostgreSQL driver but would not get it", db)
		}
	}
}
