package config

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestAnswersBecomeTheProjectTheInteractiveSetupWouldBuild(t *testing.T) {
	t.Parallel()

	answers, err := DecodeAnswers([]byte(`{
		"name": "my-site", "python": "python3", "django": "6.1.2",
		"database": "postgres-docker", "stack": "tailwind-htmx", "app": "catalog",
		"addons": ["django-ninja", "ruff", "mypy", "pre-commit"]
	}`))
	if err != nil {
		t.Fatalf("DecodeAnswers: %v", err)
	}
	p, err := answers.Project()
	if err != nil {
		t.Fatalf("Project: %v", err)
	}

	want := Project{ProjectName: "my-site", DjangoVersion: "6.1.2", DatabaseType: PostgresDocker, AppName: "catalog"}
	want.Apply(StackTailwindHTMX, []AddOn{Ninja, Ruff, Mypy, PreCommit})
	if !reflect.DeepEqual(p, want) {
		t.Errorf("Project() = %+v\nwant        %+v", p, want)
	}
	if !p.InitializeGit {
		t.Error("pre-commit without git: the rule that pre-commit needs a repository was not applied")
	}
	if !reflect.DeepEqual(p.AddedAddOns, []AddOn{Git}) {
		t.Errorf("AddedAddOns = %v, want git recorded as added for pre-commit", p.AddedAddOns)
	}
}

func TestAnAnswersFileRefusesWhatTheSetupWouldRefuse(t *testing.T) {
	t.Parallel()

	base := `"name": "shop", "python": "python3", "django": "6.1.2", "database": "sqlite", "stack": "standard", "app": "blog"`
	cases := map[string]struct {
		json string
		want string
	}{
		"unknown key":             {`{` + base + `, "addon": ["ruff"]}`, `unknown field "addon"`},
		"two values":              {`{` + base + `} {}`, "after top-level value"},
		"unknown add-on":          {`{` + base + `, "addons": ["black"]}`, `unknown add-on "black"`},
		"both api frameworks":     {`{` + base + `, "addons": ["rest-framework", "django-ninja"]}`, "pick one of Django REST Framework API and Django Ninja API"},
		"pre-release django":      {`{"name": "shop", "python": "python3", "django": "6.2a1", "database": "sqlite", "stack": "standard", "app": "blog"}`, "exact stable release"},
		"unknown database":        {`{"name": "shop", "python": "python3", "django": "6.1.2", "database": "oracle", "stack": "standard", "app": "blog"}`, `"database" must be one of`},
		"unknown stack":           {`{"name": "shop", "python": "python3", "django": "6.1.2", "database": "sqlite", "stack": "react", "app": "blog"}`, `"stack" must be one of`},
		"no python":               {`{"name": "shop", "django": "6.1.2", "database": "sqlite", "stack": "standard", "app": "blog"}`, `"python" is required`},
		"bad project name":        {`{"name": "1site", "python": "python3", "django": "6.1.2", "database": "sqlite", "stack": "standard", "app": "blog"}`, `invalid "name"`},
		"app the stack needs":     {`{"name": "shop", "python": "python3", "django": "6.1.2", "database": "sqlite", "stack": "standard"}`, `"app" is required`},
		"app shadowing a package": {`{"name": "shop", "python": "python3", "django": "6.1.2", "database": "sqlite", "stack": "standard", "app": "ninja"}`, `invalid "app"`},
	}
	for name, tc := range cases {
		answers, err := DecodeAnswers([]byte(tc.json))
		if err == nil {
			_, err = answers.Project()
		}
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: error = %v, want it to mention %q", name, err, tc.want)
		}
	}
}

func TestAnAPIOnlyProjectWithoutAnAPINeedsNoApp(t *testing.T) {
	t.Parallel()

	answers, err := DecodeAnswers([]byte(`{"name": "svc", "python": "python3", "django": "5.2.18", "database": "sqlite", "stack": "api-only", "addons": ["ruff"]}`))
	if err != nil {
		t.Fatalf("DecodeAnswers: %v", err)
	}
	if _, err := answers.Project(); err != nil {
		t.Errorf("an api-only project with no API add-on asked for an app: %v", err)
	}
}

func TestDjangoRunsOnFollowsTheSupportTableAndTheDate(t *testing.T) {
	t.Parallel()

	today := time.Date(2026, time.October, 16, 0, 0, 0, 0, time.UTC)
	afterDjango60 := time.Date(2027, time.June, 1, 0, 0, 0, 0, time.UTC)
	cases := []struct {
		django       string
		major, minor int
		day          time.Time
		want         bool
	}{
		{"6.1.2", 3, 12, today, true},
		{"6.1.2", 3, 11, today, false},
		{"5.2.18", 3, 10, today, true},
		{"6.0.9", 3, 13, today, true},
		{"6.0.9", 3, 13, afterDjango60, false},
		{"5.1.6", 3, 13, today, false},
		{"6.1.2", 4, 0, today, false},
	}
	for _, tc := range cases {
		if got := DjangoRunsOn(tc.django, tc.major, tc.minor, tc.day); got != tc.want {
			t.Errorf("DjangoRunsOn(%s, %d.%d, %s) = %v, want %v",
				tc.django, tc.major, tc.minor, tc.day.Format(time.DateOnly), got, tc.want)
		}
	}
}

func TestAnswersAcceptCommentsAndTrailingCommas(t *testing.T) {
	t.Parallel()

	answers, err := DecodeAnswers([]byte(`// my answers
{
	"name": "shop", "python": "3.13", "django": "6.1.2",
	"database": "sqlite", "stack": "standard", "app": "catalog", // the app
}`))
	if err != nil {
		t.Fatalf("DecodeAnswers: %v", err)
	}
	if answers.Name != "shop" || answers.Python != "3.13" || answers.App != "catalog" {
		t.Errorf("decoded %+v", answers)
	}
}

func TestAProjectManifestReadsBackAsTheAnswersInside(t *testing.T) {
	t.Parallel()

	answers, err := DecodeAnswers([]byte(`// Written by django-stack.
{
	"tool": "django-stack", "version": "v1.0.0", "createdAt": "2026-10-22T00:00:00Z",
	"answers": {"name": "shop", "python": "3.13", "django": "6.1.2", "database": "sqlite",
		"stack": "api-only", "addons": ["rest-framework"], "app": "catalog"},
	"resolved": {"pythonVersion": "Python 3.13.13", "moduleName": "shop"}
}`))
	if err != nil {
		t.Fatalf("DecodeAnswers: %v", err)
	}
	want := Answers{Name: "shop", Python: "3.13", Django: "6.1.2", Database: SQLite, Stack: StackAPIOnly, App: "catalog", AddOns: []string{"rest-framework"}}
	if answers.Name != want.Name || answers.Stack != want.Stack || strings.Join(answers.AddOns, ",") != "rest-framework" {
		t.Errorf("decoded %+v, want %+v", answers, want)
	}
}

func TestAManifestWithAnUnknownAnswerIsRefused(t *testing.T) {
	t.Parallel()

	_, err := DecodeAnswers([]byte(`{"tool": "django-stack", "answers": {"name": "shop", "pyhton": "3.13"}}`))
	if err == nil || !strings.Contains(err.Error(), "pyhton") {
		t.Fatalf("err = %v, want the unknown key named", err)
	}
}
