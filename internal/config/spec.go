package config

import (
	"errors"
	"fmt"
	"time"
)

type Interpreter struct {
	Major int
	Minor int
	Names PythonNames
}

type Spec struct {
	choices Choices
}

func (s Spec) Choices() Choices { return s.choices }

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

func Parse(c Choices, python Interpreter, today time.Time) (Spec, error) {
	if c.Django == "" {
		return Spec{}, &RuleError{Field: "django", Err: errors.New("no Django version was selected")}
	}
	if err := c.ValidateProjectName(python.Names); err != nil {
		return Spec{}, &RuleError{Field: "name", Value: c.Name, Err: err}
	}
	if c.NeedsAppName() && c.App == "" {
		return Spec{}, &RuleError{Field: "app", Err: errors.New("an app name is required for this stack and these add-ons")}
	}
	if err := c.ValidateAppName(python.Names); err != nil {
		return Spec{}, &RuleError{Field: "app", Value: c.App, Err: err}
	}
	if err := ValidateAddOns(c.selected()); err != nil {
		return Spec{}, &RuleError{Field: "addons", Err: err}
	}
	if python.Major != 0 && !DjangoRunsOn(c.Django, python.Major, python.Minor, today) {
		return Spec{}, &RuleError{Field: "django", Value: c.Django, Err: fmt.Errorf(
			"the selected Django release %s does not support Python %d.%d; choose another release or interpreter",
			c.Django, python.Major, python.Minor)}
	}
	return Spec{choices: c}, nil
}
