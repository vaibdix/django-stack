package config

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
)

type Project struct {
	ProjectName           string
	DjangoVersion         string
	SelectedPythonPath    string
	SelectedPythonVersion string
	AppName               string

	DatabaseType         Database
	DatabasePort         string
	DatabasePassword     string
	DatabaseRootPassword string

	Stack  Stack
	AddOns Set[AddOn]
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

func (p Project) ModuleName() string {
	return strings.ReplaceAll(p.ProjectName, "-", "_")
}

func (p Project) ValidateProjectName(names PythonNames) error {
	if p.ProjectName == "" {
		return errors.New("project name cannot be empty")
	}
	if !projectNameRegex.MatchString(p.ProjectName) {
		return fmt.Errorf("'%s' may only contain letters, numbers, hyphens, and underscores, and must start with a letter", p.ProjectName)
	}
	if err := validatePythonIdentifier(p.ModuleName()); err != nil {
		return err
	}
	if err := names.check(p.ModuleName()); err != nil {
		return err
	}
	return validateNoModuleShadowing(p.ProjectName, p.ModuleName())
}

func (p Project) ValidateAppName(names PythonNames) error {
	if p.AppName == "" {
		return nil
	}
	if err := validatePythonIdentifier(p.AppName); err != nil {
		return err
	}
	if err := names.check(p.AppName); err != nil {
		return err
	}
	if p.ProjectName != "" && p.AppName == p.ModuleName() {
		return fmt.Errorf("'%s' is the project's own package; give the app a different name", p.AppName)
	}
	return validateNoModuleShadowing(p.AppName, p.AppName)
}

func (p Project) selected() []AddOn {
	var selected []AddOn
	for _, addOn := range AddOns() {
		if p.AddOns.Has(addOn) {
			selected = append(selected, addOn)
		}
	}
	return selected
}

func (p Project) EnabledAddOns() []AddOn {
	all, _ := WithRequirements(p.selected())
	return all
}

func (p Project) AddedAddOns() []AddOn {
	_, added := WithRequirements(p.selected())
	return added
}

func (p Project) Has(addOn AddOn) bool {
	return slices.Contains(p.EnabledAddOns(), addOn)
}

func (p Project) CreateTemplates() bool {
	return p.Stack.Spec().Templates
}

func (p Project) UsesTailwind() bool {
	return p.Stack.Spec().Tailwind
}

func (p Project) UsesHTMX() bool {
	return p.Stack.Spec().HTMX
}

func (p Project) HasAPI() bool {
	return p.Has(RestFramework) || p.Has(Ninja)
}

func (p Project) NeedsAppName() bool {
	return p.CreateTemplates() || p.HasAPI() || p.UsesHTMX() || p.Has(AllAuth)
}
