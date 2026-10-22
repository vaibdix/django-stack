package config

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
)

type Database string

const (
	SQLite Database = "sqlite"

	PostgresDocker Database = "postgres-docker"

	MySQLDocker Database = "mysql-docker"
)

func (d Database) UsesDocker() bool {
	return d == PostgresDocker || d == MySQLDocker
}

func (d Database) UsesPostgres() bool {
	return d == PostgresDocker
}

func (d Database) SupportsUpsertTarget() bool {
	return d != MySQLDocker
}

type Feature string

const (
	FeatureStandardProject Feature = "Standard Django Project"
	FeatureGit             Feature = "Initialize Git"
	FeatureTailwind        Feature = "Tailwind"
	FeatureRestFramework   Feature = "REST Framework"
	FeatureNinja           Feature = "Django Ninja"
	FeatureHTMX            Feature = "HTMX"
	FeatureAllAuth         Feature = "AllAuth"
	FeatureRuff            Feature = "Ruff"
	FeatureMypy            Feature = "Mypy"
	FeaturePreCommit       Feature = "Pre-commit"
	FeatureMultiEnv        Feature = "Multi-env"
	FeaturePydanticConfig  Feature = "Pydantic Settings"
	FeatureStructlog       Feature = "Structlog"
	FeatureSentry          Feature = "Sentry"
)

type Project struct {
	ProjectName           string
	DjangoVersion         string
	SelectedPythonPath    string
	SelectedPythonVersion string
	AppName               string

	DatabaseType               Database
	PostgresDockerPort         string
	PostgresDockerPassword     string
	PostgresDockerRootPassword string
	MySQLDockerPort            string
	MySQLDockerPassword        string
	MySQLDockerRootPassword    string

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

func (p *Project) ApplyFeatures(selected []Feature) {
	p.CreateTemplates = false
	p.InitializeGit = false
	p.SetupTailwind = false
	p.SetupRestFramework = false
	p.SetupNinja = false
	p.SetupHTMX = false
	p.SetupAllAuth = false
	p.SetupRuff = false
	p.SetupMypy = false
	p.SetupPreCommit = false
	p.SetupMultiEnv = false
	p.SetupPydanticConf = false
	p.SetupStructlog = false
	p.SetupSentry = false

	for _, f := range selected {
		switch f {
		case FeatureStandardProject:
			p.CreateTemplates = true
		case FeatureGit:
			p.InitializeGit = true
		case FeatureTailwind:
			p.SetupTailwind = true
		case FeatureRestFramework:
			p.SetupRestFramework = true
		case FeatureNinja:
			p.SetupNinja = true
		case FeatureHTMX:
			p.SetupHTMX = true
		case FeatureAllAuth:
			p.SetupAllAuth = true
		case FeatureRuff:
			p.SetupRuff = true
		case FeatureMypy:
			p.SetupMypy = true
		case FeaturePreCommit:
			p.SetupPreCommit = true
		case FeatureMultiEnv:
			p.SetupMultiEnv = true
		case FeaturePydanticConfig:
			p.SetupPydanticConf = true
		case FeatureStructlog:
			p.SetupStructlog = true
		case FeatureSentry:
			p.SetupSentry = true
		}
	}

	if p.SetupPreCommit && !p.InitializeGit {
		p.InitializeGit = true
	}
}

func (p Project) HasAPI() bool {
	return p.SetupRestFramework || p.SetupNinja
}

func (p Project) NeedsAppName() bool {
	return p.CreateTemplates || p.HasAPI() || p.SetupHTMX || p.SetupAllAuth
}
