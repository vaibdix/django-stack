package config

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
)

type Interpreter struct {
	Major int
	Minor int
	Names PythonNames
}

type ProjectName struct{ name string }

func NewProjectName(name string, names PythonNames) (ProjectName, error) {
	if err := (Choices{Name: name}).ValidateProjectName(names); err != nil {
		return ProjectName{}, err
	}
	return ProjectName{name: name}, nil
}

func (n ProjectName) String() string { return n.name }

func (n ProjectName) Module() string { return strings.ReplaceAll(n.name, "-", "_") }

type AppName struct{ name string }

func NewAppName(name string, project ProjectName, names PythonNames) (AppName, error) {
	if name == "" {
		return AppName{}, errors.New("an app name cannot be empty")
	}
	if err := (Choices{Name: project.name, App: name}).ValidateAppName(names); err != nil {
		return AppName{}, err
	}
	return AppName{name: name}, nil
}

func (a AppName) String() string { return a.name }

type Option[T fmt.Stringer] struct {
	value T
	ok    bool
}

func Some[T fmt.Stringer](value T) Option[T] { return Option[T]{value: value, ok: true} }

func (o Option[T]) Get() (T, bool) { return o.value, o.ok }

func (o Option[T]) Ok() bool { return o.ok }

func (o Option[T]) String() string {
	if !o.ok {
		return ""
	}
	return o.value.String()
}

type PythonSeries struct{ major, minor int }

func (p PythonSeries) Known() bool { return p.major != 0 }

func (p PythonSeries) Minor() int { return p.minor }

func (p PythonSeries) String() string {
	if !p.Known() {
		return ""
	}
	return fmt.Sprintf("%d.%d", p.major, p.minor)
}

type DjangoRelease struct{ version string }

func NewDjangoRelease(release string, python PythonSeries, today time.Time) (DjangoRelease, error) {
	if release == "" {
		return DjangoRelease{}, errors.New("no Django version was selected")
	}
	if python.Known() && !DjangoRunsOn(release, python.major, python.minor, today) {
		return DjangoRelease{}, fmt.Errorf(
			"the selected Django release %s does not support Python %d.%d; choose another release or interpreter",
			release, python.major, python.minor)
	}
	return DjangoRelease{version: release}, nil
}

func (d DjangoRelease) String() string { return d.version }

func (d DjangoRelease) Series() string { return DjangoSeries(d.version) }

func (d DjangoRelease) SupportsNativeCSP() bool { return DjangoSupportsNativeCSP(d.version) }

func (d DjangoRelease) SupportsMailers() bool { return DjangoSupportsMailers(d.version) }

type FeatureSet struct{ enabled []AddOn }

func NewFeatureSet(selected []AddOn) (FeatureSet, []Adjustment, error) {
	if err := ValidateAddOns(selected); err != nil {
		return FeatureSet{}, nil, err
	}
	all, added := WithRequirements(selected)
	adjustments := make([]Adjustment, len(added))
	for i, addOn := range added {
		adjustments[i] = Adjustment{Added: addOn, NeededBy: RequiredBy(addOn, selected)}
	}
	return FeatureSet{enabled: all}, adjustments, nil
}

func (f FeatureSet) Has(addOn AddOn) bool { return slices.Contains(f.enabled, addOn) }

func (f FeatureSet) List() []AddOn { return slices.Clone(f.enabled) }

type Adjustment struct {
	Added    AddOn
	NeededBy []AddOn
}

func (a Adjustment) String() string {
	needers := make([]string, len(a.NeededBy))
	for i, addOn := range a.NeededBy {
		needers[i] = string(addOn)
	}
	return "Added " + string(a.Added) + ", which " + strings.Join(needers, " and ") + " needs"
}

type Spec struct {
	name        ProjectName
	python      PythonSeries
	django      DjangoRelease
	database    Database
	stack       Stack
	app         Option[AppName]
	features    FeatureSet
	adjustments []Adjustment
}

func (s Spec) Name() ProjectName { return s.name }

func (s Spec) Python() PythonSeries { return s.python }

func (s Spec) Django() DjangoRelease { return s.django }

func (s Spec) Database() Database { return s.database }

func (s Spec) Stack() Stack { return s.stack }

func (s Spec) App() Option[AppName] { return s.app }

func (s Spec) Features() FeatureSet { return s.features }

func (s Spec) Adjustments() []Adjustment { return slices.Clone(s.adjustments) }

func (s Spec) ModuleName() string { return s.name.Module() }

func (s Spec) Has(addOn AddOn) bool { return s.features.Has(addOn) }

func (s Spec) EnabledAddOns() []AddOn { return s.features.List() }

func (s Spec) AddedAddOns() []AddOn {
	var added []AddOn
	for _, adjustment := range s.adjustments {
		added = append(added, adjustment.Added)
	}
	return added
}

func (s Spec) CreateTemplates() bool { return s.stack.Spec().Templates }

func (s Spec) UsesTailwind() bool { return s.stack.Spec().Tailwind }

func (s Spec) UsesHTMX() bool { return s.stack.Spec().HTMX }

func (s Spec) APIOnly() bool { return !s.CreateTemplates() }

func (s Spec) HasAPI() bool { return s.Has(RestFramework) || s.Has(Ninja) }

func (s Spec) Choices() Choices {
	selected := slices.DeleteFunc(s.features.List(), func(a AddOn) bool { return slices.Contains(s.AddedAddOns(), a) })
	return Choices{
		Name:     s.name.String(),
		Django:   s.django.String(),
		Database: s.database,
		Stack:    s.stack,
		AddOns:   NewSet(selected...),
		App:      s.app.String(),
	}
}

func (s Spec) Answers(python string) Answers { return s.Choices().Answers(python) }

type RuleError struct {
	Field string
	Value string
	Err   error
}

func (e *RuleError) Error() string {
	if e.Value == "" {
		return e.Err.Error()
	}
	switch e.Field {
	case "name":
		return fmt.Sprintf("invalid project name '%s': %v", e.Value, e.Err)
	case "app":
		return fmt.Sprintf("invalid app name '%s': %v", e.Value, e.Err)
	default:
		return e.Err.Error()
	}
}

func (e *RuleError) Unwrap() error { return e.Err }

func Parse(c Choices, python Interpreter, today time.Time) (Spec, []Adjustment, error) {
	series := PythonSeries{major: python.Major, minor: python.Minor}
	if !c.Stack.Valid() {
		return Spec{}, nil, &RuleError{Field: "stack", Err: fmt.Errorf("unknown stack %q", c.Stack)}
	}
	if !c.Database.Valid() {
		return Spec{}, nil, &RuleError{Field: "database", Err: fmt.Errorf("unknown database %q", c.Database)}
	}
	if c.Django == "" {
		return Spec{}, nil, &RuleError{Field: "django", Err: errors.New("no Django version was selected")}
	}
	name, err := NewProjectName(c.Name, python.Names)
	if err != nil {
		return Spec{}, nil, &RuleError{Field: "name", Value: c.Name, Err: err}
	}
	spec := Spec{name: name, python: series, database: c.Database, stack: c.Stack}
	switch {
	case c.App != "":
		app, err := NewAppName(c.App, name, python.Names)
		if err != nil {
			return Spec{}, nil, &RuleError{Field: "app", Value: c.App, Err: err}
		}
		spec.app = Some(app)
	case c.NeedsAppName():
		return Spec{}, nil, &RuleError{Field: "app", Err: errors.New("an app name is required for this stack and these add-ons")}
	}
	spec.features, spec.adjustments, err = NewFeatureSet(c.selected())
	if err != nil {
		return Spec{}, nil, &RuleError{Field: "addons", Err: err}
	}
	spec.django, err = NewDjangoRelease(c.Django, series, today)
	if err != nil {
		return Spec{}, nil, &RuleError{Field: "django", Value: c.Django, Err: err}
	}
	return spec, spec.Adjustments(), nil
}
