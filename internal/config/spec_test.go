package config

import (
	"errors"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestParseRefusesADjangoReleaseThePythonCannotRun(t *testing.T) {
	t.Parallel()

	today := time.Date(2026, 10, 22, 0, 0, 0, 0, time.UTC)
	cases := []struct {
		django       string
		major, minor int
		ok           bool
	}{
		{"6.1.2", 3, 11, false},
		{"6.0.9", 3, 10, false},
		{"6.1.2", 3, 12, true},
		{"5.2.18", 3, 10, true},
		{"6.1.2", 0, 0, true},
	}
	for _, c := range cases {
		cfg := Choices{Name: "demo", Django: c.django, Database: SQLite, Stack: StackAPIOnly}
		_, _, err := Parse(cfg, Interpreter{Major: c.major, Minor: c.minor}, today)
		if c.ok && err != nil {
			t.Errorf("Django %s on 3.%d was refused: %v", c.django, c.minor, err)
		}
		if !c.ok && (err == nil || !strings.Contains(err.Error(), "does not support Python")) {
			t.Errorf("Django %s on 3.%d: err = %v, want a refusal", c.django, c.minor, err)
		}
	}
}

func TestParseNamesTheFieldThatBreaksARule(t *testing.T) {
	t.Parallel()

	python := Interpreter{Major: 3, Minor: 13, Names: PythonNames{Stdlib: []string{"telnetlib"}, Keywords: []string{"class"}}}
	cases := map[string]struct {
		cfg   Choices
		field string
		want  string
	}{
		"stdlib project":   {Choices{Name: "telnetlib", Django: "6.1.2"}, "name", "standard library"},
		"keyword app":      {Choices{Name: "shop", Django: "6.1.2", Stack: StackStandard, App: "class"}, "app", "keyword"},
		"missing app":      {Choices{Name: "shop", Django: "6.1.2", Stack: StackHTMX}, "app", "required"},
		"app is project":   {Choices{Name: "shop", Django: "6.1.2", Stack: StackStandard, App: "shop"}, "app", "own package"},
		"two apis":         {Choices{Name: "shop", Django: "6.1.2", App: "blog", AddOns: NewSet(RestFramework, Ninja)}, "addons", "not both"},
		"no django":        {Choices{Name: "shop"}, "django", "no Django"},
		"shadowed install": {Choices{Name: "shop", Django: "6.1.2", Stack: StackStandard, App: "ninja"}, "app", "django-ninja"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			cfg := c.cfg
			if cfg.Stack == "" {
				cfg.Stack = StackAPIOnly
			}
			cfg.Database = SQLite
			_, _, err := Parse(cfg, python, time.Date(2026, 10, 22, 0, 0, 0, 0, time.UTC))
			rule, ok := errors.AsType[*RuleError](err)
			if !ok || rule.Field != c.field || !strings.Contains(err.Error(), c.want) {
				t.Errorf("Parse = %v, want a %s rule error mentioning %q", err, c.field, c.want)
			}
		})
	}
}

func TestAParsedSpecKeepsTheChoices(t *testing.T) {
	t.Parallel()

	cfg := Choices{Name: "shop", Django: "6.1.2", Database: SQLite, Stack: StackHTMX, App: "catalog", AddOns: NewSet(PreCommit)}
	spec, _, err := Parse(cfg, Interpreter{}, time.Time{})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got := spec.Choices(); got.Name != cfg.Name || got.App != cfg.App || !got.Has(Git) {
		t.Errorf("Choices() = %+v, want the parsed choices with pre-commit's git", got)
	}
}

func TestEachFieldTypeRefusesBadInput(t *testing.T) {
	t.Parallel()

	names := PythonNames{Stdlib: []string{"email"}, Keywords: []string{"class"}}
	if _, err := NewProjectName("email", names); err == nil {
		t.Error("a project named after a stdlib module was accepted")
	}
	shop, err := NewProjectName("my-shop", names)
	if err != nil || shop.String() != "my-shop" || shop.Module() != "my_shop" {
		t.Fatalf("NewProjectName(my-shop) = %v, %v", shop, err)
	}
	for _, bad := range []string{"", "class", "my_shop", "1app"} {
		if _, err := NewAppName(bad, shop, names); err == nil {
			t.Errorf("NewAppName(%q) was accepted", bad)
		}
	}
	if _, err := NewDjangoRelease("6.1.2", PythonSeries{major: 3, minor: 11}, time.Date(2026, 10, 22, 0, 0, 0, 0, time.UTC)); err == nil {
		t.Error("Django 6.1 on Python 3.11 was accepted")
	}
}

func TestAnUnsetOptionPrintsNothing(t *testing.T) {
	t.Parallel()

	var none Option[AppName]
	if none.Ok() || none.String() != "" {
		t.Errorf("zero Option = %v %q", none.Ok(), none.String())
	}
	app, ok := Some(AppName{name: "catalog"}).Get()
	if !ok || app.String() != "catalog" {
		t.Errorf("Some(catalog).Get() = %v, %v", app, ok)
	}
}

func TestParseRecordsTheAddOnsItSwitchedOn(t *testing.T) {
	t.Parallel()

	cfg := Choices{Name: "shop", Django: "6.1.2", Database: SQLite, Stack: StackAPIOnly, AddOns: NewSet(PreCommit, Ruff)}
	spec, adjustments, err := Parse(cfg, Interpreter{}, time.Time{})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(adjustments) != 1 || adjustments[0].Added != Git || adjustments[0].String() != "Added git, which pre-commit needs" {
		t.Errorf("adjustments = %+v", adjustments)
	}
	if !spec.Has(Git) || !slices.Equal(spec.AddedAddOns(), []AddOn{Git}) {
		t.Errorf("the spec does not carry git: %v", spec.EnabledAddOns())
	}
	if spec.Choices().AddOns.Has(Git) {
		t.Error("Choices() reports git as picked, but Parse added it")
	}
}

func TestParseRefusesAnUnknownStackOrDatabase(t *testing.T) {
	t.Parallel()

	for field, cfg := range map[string]Choices{
		"stack":    {Name: "shop", Django: "6.1.2", Database: SQLite, Stack: "react"},
		"database": {Name: "shop", Django: "6.1.2", Database: "oracle", Stack: StackAPIOnly},
	} {
		_, _, err := Parse(cfg, Interpreter{}, time.Time{})
		if rule, ok := errors.AsType[*RuleError](err); !ok || rule.Field != field {
			t.Errorf("Parse(%+v) = %v, want a %s rule error", cfg, err, field)
		}
	}
}
