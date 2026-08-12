package scaffold

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vaibdix/django-cli/internal/config"
	"github.com/vaibdix/django-cli/internal/scaffold/command"
	"github.com/vaibdix/django-cli/internal/scaffold/pyenv"
)

func manifestFor(t *testing.T, cfg config.Project) (string, stackManifest) {
	t.Helper()

	projectPath := t.TempDir()
	e := &Env{
		Cfg:      cfg,
		Reporter: NopReporter{},
		Clock:    func() time.Time { return time.Date(2026, 8, 12, 9, 30, 0, 0, time.UTC) },
	}
	if err := e.writeStackManifest(projectPath); err != nil {
		t.Fatalf("writeStackManifest: %v", err)
	}

	raw, err := os.ReadFile(filepath.Join(projectPath, StackManifestFile))
	if err != nil {
		t.Fatalf("reading the manifest: %v", err)
	}
	content := string(raw)

	// .jsonc allows comments, so strip the preamble before decoding.
	start := strings.Index(content, "{")
	if start == -1 {
		t.Fatalf("the manifest contains no JSON object:\n%s", content)
	}
	body := content[start:]
	var decoded stackManifest
	if err := json.Unmarshal([]byte(body), &decoded); err != nil {
		t.Fatalf("the manifest is not valid JSON once the comments are stripped: %v\n%s", err, content)
	}
	return content, decoded
}

func TestManifestRecordsEveryChoiceTheRunWasGiven(t *testing.T) {
	t.Parallel()

	cfg := config.Project{
		ProjectName: "my-shop", AppName: "catalog",
		DjangoVersion: "6.1", SelectedPythonVersion: "Python 3.13.13",
		SelectedPythonPath: "/usr/bin/python3.13", DatabaseType: config.PostgresDocker,
	}
	cfg.ApplyFeatures(append(config.StackTailwindHTMX.Features(),
		config.FeatureRestFramework, config.FeatureRuff))

	_, m := manifestFor(t, cfg)

	checks := map[string]struct{ got, want string }{
		"projectName": {m.ProjectName, "my-shop"},
		"moduleName":  {m.ModuleName, "my_shop"},
		"django":      {m.Django, "6.1"},
		"database":    {m.Database.Engine, "postgres-docker"},
		"dbImage":     {m.Database.Image, "postgres:18"},
		"stack":       {m.Stack, "tailwind-htmx"},
		"appName":     {m.AppName, "catalog"},
		"createdAt":   {m.CreatedAt, "2026-08-12T09:30:00Z"},
		"tool":        {m.Tool, "django-stack"},
		"interpreter": {m.Python.Interpreter, "/usr/bin/python3.13"},
	}
	for field, c := range checks {
		if c.got != c.want {
			t.Errorf("%s = %q, want %q", field, c.got, c.want)
		}
	}

	if want := []string{"rest-framework", "ruff"}; strings.Join(m.AddOns, ",") != strings.Join(want, ",") {
		t.Errorf("addons = %v, want %v", m.AddOns, want)
	}
}

func TestManifestReportsTheStackNotItsUnderlyingFlags(t *testing.T) {
	t.Parallel()

	for _, stack := range config.AllStacks() {
		cfg := config.Project{ProjectName: "demo", DjangoVersion: "6.1", DatabaseType: config.SQLite}
		cfg.ApplyFeatures(stack.Features())

		_, m := manifestFor(t, cfg)
		if m.Stack != string(stack) {
			t.Errorf("a %s project reports stack %q", stack, m.Stack)
		}
		for _, addOn := range m.AddOns {
			if addOn == "tailwind" || addOn == "htmx" || addOn == "standard" {
				t.Errorf("%s leaked a stack feature into the add-on list: %v", stack, m.AddOns)
			}
		}
	}
}

func TestManifestNeverRecordsACredential(t *testing.T) {
	t.Parallel()

	cfg := config.Project{
		ProjectName: "demo", DjangoVersion: "6.1", DatabaseType: config.PostgresDocker,
		PostgresDockerPassword: "sup3rs3cret-do-not-write-me",
		MySQLDockerPassword:    "another-secret",
	}
	cfg.ApplyFeatures(append(config.StackStandard.Features(),
		config.FeatureSentry, config.FeaturePydanticConfig))

	content, _ := manifestFor(t, cfg)
	for _, secret := range []string{"sup3rs3cret-do-not-write-me", "another-secret"} {
		if strings.Contains(content, secret) {
			t.Errorf("the manifest leaked a generated password:\n%s", content)
		}
	}

	// A Sentry DSN is a credential too: it is a write token for the project's
	// error stream, and this file is meant to be committable.
	for _, name := range []string{"SENTRY_DSN", "SECRET_KEY", "DB_PASSWORD"} {
		if strings.Contains(content, name) {
			t.Errorf("the manifest names the credential %s; it belongs in .env only:\n%s", name, content)
		}
	}
}

func TestManifestIsValidJSONCWithACommentPreamble(t *testing.T) {
	t.Parallel()

	cfg := config.Project{ProjectName: "demo", DjangoVersion: "6.1", DatabaseType: config.SQLite}
	cfg.ApplyFeatures(config.StackStandard.Features())

	content, _ := manifestFor(t, cfg)
	if !strings.HasPrefix(content, "//") {
		t.Errorf("the manifest should open with a comment explaining itself:\n%s", content)
	}
	if !strings.HasSuffix(content, "}\n") {
		t.Errorf("the manifest should end with a newline after the object:\n%q", content[len(content)-10:])
	}
}

func TestManifestIsReadableRatherThanHTMLEscaped(t *testing.T) {
	t.Parallel()

	cfg := config.Project{
		ProjectName: "demo", DjangoVersion: "6.1",
		SelectedPythonVersion: "Python 3.13.13", DatabaseType: config.SQLite,
	}
	cfg.ApplyFeatures(config.StackStandard.Features())

	content, m := manifestFor(t, cfg)

	for _, escape := range []string{`\u003e`, `\u003c`, `\u0026`} {
		if strings.Contains(content, escape) {
			t.Errorf("the manifest is HTML-escaped, so a reader sees %s instead of the character:\n%s",
				escape, content)
		}
	}
	if m.Python.RequiresPython != ">=3.13" {
		t.Errorf("requiresPython = %q, want >=3.13", m.Python.RequiresPython)
	}
}

func TestManifestRecordsDirectDependenciesAtTheirResolvedVersions(t *testing.T) {
	t.Parallel()

	projectPath := t.TempDir()
	fake := command.NewFake()
	name, args := pyenv.FreezeCommand(projectPath)
	fake.Responses[command.Call{Name: name, Args: args}.String()] = []byte(
		"asgiref==3.9.1\nDjango==6.1\ndjangorestframework==3.18.0\nsqlparse==0.5.5\nruff==0.16.2\n")

	cfg := config.Project{ProjectName: "demo", DjangoVersion: "6.1", DatabaseType: config.SQLite}
	cfg.ApplyFeatures(config.StackStandard.Features())

	e := &Env{Cfg: cfg, Runner: fake, Reporter: NopReporter{}}
	e.deps.addRuntime("django==6.1", PkgRestFramework)
	e.deps.addDev(PkgRuff)

	m := e.buildStackManifest(projectPath)

	if got := m.Dependencies.Python["djangorestframework"]; got != "3.18.0" {
		t.Errorf("djangorestframework = %q, want the resolved 3.18.0", got)
	}
	if got := m.Dependencies.Python["django"]; got != "6.1" {
		t.Errorf("django = %q, want 6.1", got)
	}
	if got := m.Dependencies.PythonDev["ruff"]; got != "0.16.2" {
		t.Errorf("ruff = %q, want 0.16.2", got)
	}

	for _, transitive := range []string{"asgiref", "sqlparse"} {
		if _, listed := m.Dependencies.Python[transitive]; listed {
			t.Errorf("%s is a transitive dependency; only what the project asked for belongs here", transitive)
		}
	}
}

func TestManifestSeparatesRuntimeFromDevDependencies(t *testing.T) {
	t.Parallel()

	projectPath := t.TempDir()
	cfg := config.Project{ProjectName: "demo", DjangoVersion: "6.1", DatabaseType: config.SQLite}
	cfg.ApplyFeatures(config.StackStandard.Features())

	e := &Env{Cfg: cfg, Runner: command.NewFake(), Reporter: NopReporter{}}
	e.deps.addRuntime(PkgRestFramework)
	e.deps.addDev(PkgMypy, PkgRuff)

	m := e.buildStackManifest(projectPath)

	if _, ok := m.Dependencies.Python["mypy"]; ok {
		t.Error("mypy is a dev tool but was listed as a runtime dependency")
	}
	if _, ok := m.Dependencies.PythonDev["djangorestframework"]; ok {
		t.Error("djangorestframework is a runtime dependency but was listed as a dev tool")
	}
}
