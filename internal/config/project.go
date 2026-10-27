package config

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"
)

type Choices struct {
	Name     string
	Django   string
	Database Database
	Stack    Stack
	AddOns   Set[AddOn]
	App      string
}

type Resolved struct {
	PythonPath    string
	PythonVersion string
	Database      DatabaseCredentials
	SecretKey     string
	Generated     time.Time
	IndexDate     time.Time
}

type DatabaseCredentials struct {
	Port         string
	Password     string
	RootPassword string
}

var (
	pythonIdentifierRegex = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

	projectNameRegex = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]*$`)
)

type PythonNames struct {
	Stdlib   []string
	Keywords []string
}

func (n PythonNames) check(moduleName string) error {
	if slices.Contains(n.Keywords, moduleName) {
		return fmt.Errorf("'%s' is a Python keyword", moduleName)
	}
	if slices.Contains(n.Stdlib, moduleName) {
		return fmt.Errorf(
			"'%s' is a module in Python's standard library; a package with that name would shadow it "+
				"and Django refuses to create it. Pick another name, for example '%s_app'", moduleName, moduleName)
	}
	return nil
}

func validatePythonIdentifier(name string) error {
	if name == "" {
		return errors.New("identifier cannot be empty")
	}
	if !pythonIdentifierRegex.MatchString(name) {
		return fmt.Errorf("'%s' is not a valid Python identifier", name)
	}
	return nil
}

func (p Choices) ModuleName() string {
	return strings.ReplaceAll(p.Name, "-", "_")
}

func (p Choices) ValidateProjectName(names PythonNames) error {
	if p.Name == "" {
		return errors.New("project name cannot be empty")
	}
	if !projectNameRegex.MatchString(p.Name) {
		return fmt.Errorf("'%s' may only contain letters, numbers, hyphens, and underscores, and must start with a letter", p.Name)
	}
	if err := validatePythonIdentifier(p.ModuleName()); err != nil {
		return err
	}
	if err := names.check(p.ModuleName()); err != nil {
		return err
	}
	return validateNoModuleShadowing(p.Name, p.ModuleName())
}

func (p Choices) ValidateAppName(names PythonNames) error {
	if p.App == "" {
		return nil
	}
	if err := validatePythonIdentifier(p.App); err != nil {
		return err
	}
	if err := names.check(p.App); err != nil {
		return err
	}
	if p.Name != "" && p.App == p.ModuleName() {
		return fmt.Errorf("'%s' is the project's own package; give the app a different name", p.App)
	}
	return validateNoModuleShadowing(p.App, p.App)
}

func (p Choices) selected() []AddOn {
	var selected []AddOn
	for _, addOn := range AddOns() {
		if p.AddOns.Has(addOn) {
			selected = append(selected, addOn)
		}
	}
	return selected
}

func (p Choices) EnabledAddOns() []AddOn {
	all, _ := WithRequirements(p.selected())
	return all
}

func (p Choices) AddedAddOns() []AddOn {
	_, added := WithRequirements(p.selected())
	return added
}

func (p Choices) Has(addOn AddOn) bool {
	return slices.Contains(p.EnabledAddOns(), addOn)
}

func (p Choices) CreateTemplates() bool {
	return p.Stack.Spec().Templates
}

func (p Choices) UsesTailwind() bool {
	return p.Stack.Spec().Tailwind
}

func (p Choices) UsesHTMX() bool {
	return p.Stack.Spec().HTMX
}

func (p Choices) HasAPI() bool {
	return p.Has(RestFramework) || p.Has(Ninja)
}

func (p Choices) NeedsAppName() bool {
	return p.CreateTemplates() || p.HasAPI() || p.UsesHTMX() || p.Has(AllAuth)
}
