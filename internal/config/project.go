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

	CreateTemplates    bool
	InitializeGit      bool
	SetupTailwind      bool
	SetupRestFramework bool
	SetupNinja         bool
	SetupHTMX          bool
	SetupAllAuth       bool
	SetupRuff          bool
	SetupMypy          bool
	SetupPreCommit     bool
	SetupMultiEnv      bool
	SetupPydanticConf  bool
	SetupStructlog     bool
	SetupSentry        bool

	AddedAddOns []AddOn
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

func (p *Project) Apply(stack Stack, selected []AddOn) {
	spec := stack.Spec()
	p.CreateTemplates, p.SetupTailwind, p.SetupHTMX = spec.Templates, spec.Tailwind, spec.HTMX

	all, added := WithRequirements(selected)
	p.AddedAddOns = added
	has := func(a AddOn) bool { return slices.Contains(all, a) }
	p.InitializeGit = has(Git)
	p.SetupRestFramework = has(RestFramework)
	p.SetupNinja = has(Ninja)
	p.SetupAllAuth = has(AllAuth)
	p.SetupRuff = has(Ruff)
	p.SetupMypy = has(Mypy)
	p.SetupPreCommit = has(PreCommit)
	p.SetupMultiEnv = has(MultiEnv)
	p.SetupPydanticConf = has(Pydantic)
	p.SetupStructlog = has(Structlog)
	p.SetupSentry = has(Sentry)
}

func (p Project) HasAPI() bool {
	return p.SetupRestFramework || p.SetupNinja
}

func (p Project) NeedsAppName() bool {
	return p.CreateTemplates || p.HasAPI() || p.SetupHTMX || p.SetupAllAuth
}
