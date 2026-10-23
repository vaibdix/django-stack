package config

import (
	"fmt"
	"slices"
	"strings"
)

type AddOn string

const (
	Git           AddOn = "git"
	RestFramework AddOn = "rest-framework"
	Ninja         AddOn = "django-ninja"
	AllAuth       AddOn = "allauth"
	Ruff          AddOn = "ruff"
	Mypy          AddOn = "mypy"
	PreCommit     AddOn = "pre-commit"
	MultiEnv      AddOn = "multi-env"
	Pydantic      AddOn = "pydantic"
	Structlog     AddOn = "structlog"
	Sentry        AddOn = "sentry"
)

type AddOnSpec struct {
	Name      AddOn
	Label     string
	Requires  []AddOn
	Conflicts []AddOn
}

var addOns = []AddOnSpec{
	{Name: Git, Label: "Initialize Git Repository"},
	{Name: RestFramework, Label: "Django REST Framework API", Conflicts: []AddOn{Ninja}},
	{Name: Ninja, Label: "Django Ninja API", Conflicts: []AddOn{RestFramework}},
	{Name: AllAuth, Label: "Django AllAuth (authentication)"},
	{Name: Ruff, Label: "Ruff (Linter + Formatter)"},
	{Name: Mypy, Label: "Mypy (Type Checking)"},
	{Name: PreCommit, Label: "Pre-commit Hooks (requires Git)", Requires: []AddOn{Git}},
	{Name: MultiEnv, Label: "Multi-env Settings (dev/staging/prod)"},
	{Name: Pydantic, Label: "Typed Python (pydantic)"},
	{Name: Structlog, Label: "Structured Logging (structlog)"},
	{Name: Sentry, Label: "Error Tracking (Sentry)"},
}

func AddOns() []AddOn {
	names := make([]AddOn, len(addOns))
	for i, spec := range addOns {
		names[i] = spec.Name
	}
	return names
}

func (a AddOn) Valid() bool {
	return slices.ContainsFunc(addOns, func(spec AddOnSpec) bool { return spec.Name == a })
}

func (a AddOn) Spec() AddOnSpec {
	i := slices.IndexFunc(addOns, func(spec AddOnSpec) bool { return spec.Name == a })
	if i < 0 {
		return AddOnSpec{Name: a, Label: string(a)}
	}
	return addOns[i]
}

func ParseAddOns(names []string) ([]AddOn, error) {
	selected := make([]AddOn, 0, len(names))
	for _, name := range names {
		addOn := AddOn(name)
		if !addOn.Valid() {
			return nil, fmt.Errorf("unknown add-on %q; the add-ons are %s", name, joinAddOns())
		}
		selected = append(selected, addOn)
	}
	return selected, nil
}

func ValidateAddOns(selected []AddOn) error {
	for i, a := range selected {
		for _, b := range selected[i+1:] {
			if slices.Contains(a.Spec().Conflicts, b) {
				return fmt.Errorf("pick one of %s and %s, not both", a.Spec().Label, b.Spec().Label)
			}
		}
	}
	return nil
}

func WithRequirements(selected []AddOn) (all, added []AddOn) {
	all = slices.Clone(selected)
	for i := 0; i < len(all); i++ {
		for _, required := range all[i].Spec().Requires {
			if !slices.Contains(all, required) {
				all = append(all, required)
				added = append(added, required)
			}
		}
	}
	return all, added
}

func RequiredBy(addOn AddOn, selected []AddOn) []AddOn {
	var by []AddOn
	for _, a := range selected {
		if slices.Contains(a.Spec().Requires, addOn) {
			by = append(by, a)
		}
	}
	return by
}

func joinAddOns() string {
	names := make([]string, 0, len(addOns))
	for _, spec := range addOns {
		names = append(names, string(spec.Name))
	}
	slices.Sort(names)
	return strings.Join(names, ", ")
}
