package config

import (
	"slices"
	"strings"
	"testing"
	"time"
)

func FuzzValidateProjectNameImpliesValidModule(f *testing.F) {
	seeds := []string{
		"myproject", "my-project", "my_project", "a", "A1",
		"my--project", "project-", "-project", "", "1project",
		"class", "import-", "for", "None",
		"my-class", "my-import",
		"σigma", "my project", "my.project", "my/project",
		strings.Repeat("a", 300),
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, name string) {
		p := Choices{Name: name}
		if p.ValidateProjectName(PythonNames{}) != nil {
			return
		}

		module := p.ModuleName()

		if err := validatePythonIdentifier(module); err != nil {
			t.Errorf("ValidateProjectName accepted %q, but its module name %q is not a valid Python identifier: %v",
				name, module, err)
		}
		if strings.Contains(module, "-") {
			t.Errorf("module name %q derived from %q still contains a hyphen", module, name)
		}
		if module == "" {
			t.Errorf("project name %q produced an empty module name", name)
		}
	})
}

func FuzzModuleNameIsIdempotent(f *testing.F) {
	for _, s := range []string{"my-project", "a-b-c", "already_fine", "-", "--"} {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, name string) {
		once := Choices{Name: name}.ModuleName()
		twice := Choices{Name: once}.ModuleName()
		if once != twice {
			t.Errorf("ModuleName is not idempotent for %q: %q then %q", name, once, twice)
		}
	})
}

func FuzzValidateAppName(f *testing.F) {
	for _, s := range []string{"", "catalog", "my-app", "class", "1app", "_private"} {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, name string) {
		err := Choices{App: name}.ValidateAppName(PythonNames{})

		if name == "" {
			if err != nil {
				t.Errorf("an empty app name must be valid, got %v", err)
			}
			return
		}
		if err == nil {
			if idErr := validatePythonIdentifier(name); idErr != nil {
				t.Errorf("ValidateAppName accepted %q which is not a Python identifier: %v", name, idErr)
			}
		}
	})
}

func FuzzAParsedSpecRoundTripsThroughItsAnswers(f *testing.F) {
	f.Add("shop", "catalog", "htmx", uint16(0b1010))
	f.Add("my-project", "", "api-only", uint16(0))
	f.Add("class", "for", "tailwind", uint16(0xffff))
	f.Add("shop", "shop", "standard", uint16(0b110))

	f.Fuzz(func(t *testing.T, name, app, stack string, mask uint16) {
		all := AddOns()
		var picked []AddOn
		for i, addOn := range all {
			if mask&(1<<i) != 0 {
				picked = append(picked, addOn)
			}
		}
		cfg := Choices{Name: name, App: app, Django: "6.1.2", Database: SQLite, Stack: Stack(stack), AddOns: NewSet(picked...)}
		spec, err := Parse(cfg, Interpreter{Major: 3, Minor: 13}, time.Date(2026, 10, 22, 0, 0, 0, 0, time.UTC))
		if err != nil || !cfg.Stack.Valid() {
			return
		}
		again, err := spec.Choices().Answers("3.13").Choices()
		if err != nil {
			t.Fatalf("a parsed spec's answers were refused: %v", err)
		}
		if again.Name != cfg.Name || again.App != cfg.App || again.Stack != cfg.Stack ||
			!slices.Equal(slices.Sorted(slices.Values(again.EnabledAddOns())), slices.Sorted(slices.Values(spec.Choices().EnabledAddOns()))) {
			t.Errorf("round trip changed the choices: %+v became %+v", spec.Choices(), again)
		}
	})
}
