package config

import (
	"errors"
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
		cfg := Choices{Name: "demo", Django: c.django, Database: SQLite}
		_, err := Parse(cfg, Interpreter{Major: c.major, Minor: c.minor}, today)
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
			_, err := Parse(c.cfg, python, time.Date(2026, 10, 22, 0, 0, 0, 0, time.UTC))
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
	spec, err := Parse(cfg, Interpreter{}, time.Time{})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got := spec.Choices(); got.Name != cfg.Name || got.App != cfg.App || !got.Has(Git) {
		t.Errorf("Choices() = %+v, want the parsed choices with pre-commit's git", got)
	}
}
