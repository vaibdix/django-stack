package scaffold

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/vaibdix/django-stack/internal/command/commandtest"
	"github.com/vaibdix/django-stack/internal/config"
	"github.com/vaibdix/django-stack/internal/pyenv"
	"github.com/vaibdix/django-stack/internal/report"
)

func manifestFor(t *testing.T, cfg config.Choices, resolved config.Resolved) (string, stackManifest) {
	t.Helper()

	projectPath := t.TempDir()
	e := &Env{
		cfg:      mustSpec(cfg),
		resolved: resolved,
		runner:   commandtest.NewFake(),
		reporter: report.Nop{},
		clock:    func() time.Time { return time.Date(2026, 8, 12, 9, 30, 0, 0, time.UTC) },
	}
	if cfg.UsesTailwind() {
		e.node = pnpm
		packageJSON := `{"dependencies": {"tailwindcss": "^4.1.0", "@tailwindcss/cli": "^4.1.0"}}`
		if err := os.WriteFile(filepath.Join(projectPath, "package.json"), []byte(packageJSON), 0o644); err != nil {
			t.Fatalf("writing package.json: %v", err)
		}
	}
	e.dir = projectPath
	if err := e.writeStackManifest(t.Context()); err != nil {
		t.Fatalf("writeStackManifest: %v", err)
	}

	raw, err := os.ReadFile(filepath.Join(projectPath, stackManifestFile))
	if err != nil {
		t.Fatalf("reading the manifest: %v", err)
	}
	content := string(raw)

	var decoded stackManifest
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("the manifest is not valid JSON: %v\n%s", err, content)
	}
	return content, decoded
}

func TestManifestRecordsEveryChoiceTheRunWasGiven(t *testing.T) {
	t.Parallel()

	cfg := config.Choices{Name: "my-shop", App: "catalog", Django: "6.1", Database: config.PostgresDocker}
	resolved := config.Resolved{PythonVersion: "Python 3.13.13", PythonPath: "/usr/bin/python3.13"}
	cfg.Stack, cfg.AddOns = config.StackTailwindHTMX, config.NewSet(config.RestFramework, config.Ruff)

	_, m := manifestFor(t, cfg, resolved)

	checks := map[string]struct{ got, want string }{
		"projectName":   {m.Answers.Name, "my-shop"},
		"moduleName":    {m.Resolved.ModuleName, "my_shop"},
		"django":        {m.Answers.Django, "6.1"},
		"database":      {string(m.Answers.Database), "postgres-docker"},
		"dbImage":       {m.Resolved.DatabaseImage, "postgres:18"},
		"stack":         {string(m.Answers.Stack), "tailwind-htmx"},
		"appName":       {m.Answers.App, "catalog"},
		"createdAt":     {m.CreatedAt, "2026-08-12T09:30:00Z"},
		"tool":          {m.Tool, "django-stack"},
		"pythonVersion": {m.Resolved.PythonVersion, "Python 3.13.13"},
		"python":        {m.Answers.Python, "3.13"},
	}
	for field, c := range checks {
		if c.got != c.want {
			t.Errorf("%s = %q, want %q", field, c.got, c.want)
		}
	}

	if want := []string{"rest-framework", "ruff"}; strings.Join(m.Answers.AddOns, ",") != strings.Join(want, ",") {
		t.Errorf("addons = %v, want %v", m.Answers.AddOns, want)
	}
}

func TestManifestReportsTheStackNotItsUnderlyingFlags(t *testing.T) {
	t.Parallel()

	for _, stack := range config.AllStacks() {
		cfg := config.Choices{Name: "demo", Django: "6.1", Database: config.SQLite}
		cfg.Stack, cfg.AddOns = stack, nil

		_, m := manifestFor(t, cfg, config.Resolved{})
		if m.Answers.Stack != stack {
			t.Errorf("a %s project reports stack %q", stack, m.Answers.Stack)
		}
		for _, addOn := range m.Answers.AddOns {
			if addOn == "tailwind" || addOn == "htmx" || addOn == "standard" {
				t.Errorf("%s leaked a stack feature into the add-on list: %v", stack, m.Answers.AddOns)
			}
		}
	}
}

func TestManifestNeverRecordsACredential(t *testing.T) {
	t.Parallel()

	cfg := config.Choices{Name: "demo", Django: "6.1", Database: config.PostgresDocker}
	resolved := config.Resolved{Database: config.DatabaseCredentials{Password: "sup3rs3cret-do-not-write-me", RootPassword: "root-secret"}}
	cfg.Stack, cfg.AddOns = config.StackStandard, config.NewSet(config.Sentry, config.Pydantic)

	content, _ := manifestFor(t, cfg, resolved)
	for _, secret := range []string{"sup3rs3cret-do-not-write-me", "root-secret"} {
		if strings.Contains(content, secret) {
			t.Errorf("the manifest leaked a generated password:\n%s", content)
		}
	}

	for _, name := range []string{"SENTRY_DSN", "SECRET_KEY", "DB_PASSWORD"} {
		if strings.Contains(content, name) {
			t.Errorf("the manifest names the credential %s; it belongs in .env only:\n%s", name, content)
		}
	}
}

func TestManifestIsPlainJSON(t *testing.T) {
	t.Parallel()

	cfg := config.Choices{Name: "demo", Django: "6.1", Database: config.SQLite}
	cfg.Stack, cfg.AddOns = config.StackStandard, nil

	content, _ := manifestFor(t, cfg, config.Resolved{})
	if !strings.HasPrefix(content, "{") || !json.Valid([]byte(content)) {
		t.Errorf("%s must be plain JSON that any JSON reader accepts:\n%s", stackManifestFile, content)
	}
	if !strings.HasSuffix(content, "}\n") {
		t.Errorf("the manifest should end with a newline after the object:\n%q", content[len(content)-10:])
	}
}

func TestManifestIsReadableRatherThanHTMLEscaped(t *testing.T) {
	t.Parallel()

	cfg := config.Choices{Name: "demo", Django: "6.1", Database: config.SQLite}
	resolved := config.Resolved{PythonVersion: "Python 3.13.13"}
	cfg.Stack, cfg.AddOns = config.StackStandard, nil

	content, m := manifestFor(t, cfg, resolved)

	for _, escape := range []string{`\u003e`, `\u003c`, `\u0026`} {
		if strings.Contains(content, escape) {
			t.Errorf("the manifest is HTML-escaped, so a reader sees %s instead of the character:\n%s",
				escape, content)
		}
	}
	if m.Resolved.RequiresPython != ">=3.13" {
		t.Errorf("requiresPython = %q, want >=3.13", m.Resolved.RequiresPython)
	}
}

func TestManifestRecordsDirectDependenciesAtTheirResolvedVersions(t *testing.T) {
	t.Parallel()

	projectPath := t.TempDir()
	fake := commandtest.NewFake()
	name, args := pyenv.FreezeCommand(pyenv.Pip, projectPath)
	fake.Responses[commandtest.Call{Name: name, Args: args}.String()] = []byte(
		"asgiref==3.9.1\nDjango==6.1\ndjangorestframework==3.18.0\nsqlparse==0.5.5\nruff==0.16.2\n")

	cfg := config.Choices{Name: "demo", Django: "6.1", Database: config.SQLite}
	cfg.Stack, cfg.AddOns = config.StackStandard, nil

	e := &Env{cfg: mustSpec(cfg), runner: fake, reporter: report.Nop{}, clock: reviewClock}
	e.packages.addRuntime("django==6.1", pkgRestFramework)
	e.packages.addDev(pkgRuff)

	m, err := e.buildStackManifest(t.Context(), projectPath)
	if err != nil {
		t.Fatalf("buildStackManifest: %v", err)
	}

	if got := m.Resolved.Dependencies.Python["djangorestframework"]; got != "3.18.0" {
		t.Errorf("djangorestframework = %q, want the resolved 3.18.0", got)
	}
	if got := m.Resolved.Dependencies.Python["django"]; got != "6.1" {
		t.Errorf("django = %q, want 6.1", got)
	}
	if got := m.Resolved.Dependencies.PythonDev["ruff"]; got != "0.16.2" {
		t.Errorf("ruff = %q, want 0.16.2", got)
	}

	for _, transitive := range []string{"asgiref", "sqlparse"} {
		if _, listed := m.Resolved.Dependencies.Python[transitive]; listed {
			t.Errorf("%s is a transitive dependency; only what the project asked for belongs here", transitive)
		}
	}
}

func TestManifestSeparatesRuntimeFromDevDependencies(t *testing.T) {
	t.Parallel()

	projectPath := t.TempDir()
	cfg := config.Choices{Name: "demo", Django: "6.1", Database: config.SQLite}
	cfg.Stack, cfg.AddOns = config.StackStandard, nil

	fake := commandtest.NewFake()
	name, args := pyenv.FreezeCommand(pyenv.Pip, projectPath)
	fake.Responses[commandtest.Call{Name: name, Args: args}.String()] = []byte(
		"djangorestframework==3.18.0\nmypy==2.4.0\nruff==0.17.1\n")
	e := &Env{cfg: mustSpec(cfg), runner: fake, reporter: report.Nop{}, clock: reviewClock}
	e.packages.addRuntime(pkgRestFramework)
	e.packages.addDev(pkgMypy, pkgRuff)

	m, err := e.buildStackManifest(t.Context(), projectPath)
	if err != nil {
		t.Fatalf("buildStackManifest: %v", err)
	}

	if _, ok := m.Resolved.Dependencies.Python["mypy"]; ok {
		t.Error("mypy is a dev tool but was listed as a runtime dependency")
	}
	if _, ok := m.Resolved.Dependencies.PythonDev["djangorestframework"]; ok {
		t.Error("djangorestframework is a runtime dependency but was listed as a dev tool")
	}
}

func TestManifestNeverRecordsTheGeneratingMachine(t *testing.T) {
	t.Parallel()

	cfg := config.Choices{Name: "my-shop", Django: "6.1", Database: config.SQLite}
	resolved := config.Resolved{
		PythonVersion: "Python 3.13.13",
		PythonPath:    "/Users/someone/.local/share/uv/python/cpython-3.13/bin/python3.13",
	}

	content, _ := manifestFor(t, cfg, resolved)

	for _, leak := range []string{"/Users/", "/home/", "interpreter"} {
		if strings.Contains(content, leak) {
			t.Errorf("the manifest is committed to version control and leaks %q:\n%s", leak, content)
		}
	}
}

func TestTheManifestReadsBackAsTheAnswersThatMadeTheProject(t *testing.T) {
	t.Parallel()

	cfg := config.Choices{Name: "my-shop", App: "catalog", Django: "6.1.2", Database: config.MySQLDocker}
	resolved := config.Resolved{PythonVersion: "Python 3.13.13"}
	addOns := []config.AddOn{config.Ninja, config.Mypy, config.PreCommit}
	cfg.Stack, cfg.AddOns = config.StackHTMX, config.NewSet(addOns...)

	content, _ := manifestFor(t, cfg, resolved)
	answers, err := config.DecodeAnswers([]byte(content))
	if err != nil {
		t.Fatalf("the manifest is not an answers file: %v\n%s", err, content)
	}
	replayed, err := answers.Choices()
	if err != nil {
		t.Fatalf("the manifest's answers are refused: %v", err)
	}

	want := config.Choices{Name: cfg.Name, App: cfg.App, Django: cfg.Django, Database: cfg.Database}
	want.Stack, want.AddOns = config.StackHTMX, config.NewSet(append(addOns, config.Git)...)
	if !reflect.DeepEqual(replayed, want) {
		t.Errorf("replayed project = %+v\nwant              %+v", replayed, want)
	}
	if answers.Python != "3.13" {
		t.Errorf("python = %q, want the series 3.13", answers.Python)
	}
}

func TestTheManifestRecordsAnAddOnThatWasAddedForAnother(t *testing.T) {
	t.Parallel()

	cfg := config.Choices{Name: "demo", App: "catalog", Django: "6.1", Database: config.SQLite}
	cfg.Stack, cfg.AddOns = config.StackStandard, config.NewSet(config.PreCommit)
	_, manifest := manifestFor(t, cfg, config.Resolved{})
	if !slices.Equal(manifest.Answers.AddOns, []string{"git", "pre-commit"}) {
		t.Errorf("answers.addons = %v, want git and pre-commit", manifest.Answers.AddOns)
	}
	if !slices.Equal(manifest.Resolved.AddedAddOns, []config.AddOn{config.Git}) {
		t.Errorf("resolved.addedAddOns = %v, want [git]", manifest.Resolved.AddedAddOns)
	}

	cfg.Stack, cfg.AddOns = config.StackStandard, config.NewSet(config.Git, config.PreCommit)
	if content, _ := manifestFor(t, cfg, config.Resolved{}); strings.Contains(content, "addedAddOns") {
		t.Errorf("git was picked, yet the manifest says it was added:\n%s", content)
	}
}
