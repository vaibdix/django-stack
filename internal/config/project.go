package config

import (
	"fmt"
	"regexp"
	"strings"
)

type Database string

const (
	SQLite Database = "sqlite"

	PostgresDocker Database = "postgres-docker"

	PostgresString Database = "postgres-string"

	MySQLDocker Database = "mysql-docker"
)

func (d Database) UsesDocker() bool {
	return d == PostgresDocker || d == MySQLDocker
}

func (d Database) UsesPostgres() bool {
	return d == PostgresDocker || d == PostgresString
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

	DatabaseType           Database
	PostgresDockerPort     string
	PostgresDockerPassword string
	MySQLDockerPort        string
	MySQLDockerPassword    string

	CreateTemplates    bool
	CreateAppTemplates bool
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
	UseGlobalTemplates bool
	StartDevServer     bool
}

var (
	pythonIdentifierRegex = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

	projectNameRegex = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]*$`)
)

var pythonReservedWords = map[string]bool{
	"False": true, "None": true, "True": true, "and": true, "as": true,
	"assert": true, "async": true, "await": true, "break": true, "class": true,
	"continue": true, "def": true, "del": true, "elif": true, "else": true,
	"except": true, "finally": true, "for": true, "from": true, "global": true,
	"if": true, "import": true, "in": true, "is": true, "lambda": true,
	"nonlocal": true, "not": true, "or": true, "pass": true, "raise": true,
	"return": true, "try": true, "while": true, "with": true, "yield": true,
}

func ValidatePythonIdentifier(name string) error {
	if name == "" {
		return fmt.Errorf("identifier cannot be empty")
	}
	if pythonReservedWords[name] {
		return fmt.Errorf("'%s' is a Python reserved word", name)
	}
	if !pythonIdentifierRegex.MatchString(name) {
		return fmt.Errorf("'%s' is not a valid Python identifier", name)
	}
	return nil
}

func (p Project) ModuleName() string {
	return strings.ReplaceAll(p.ProjectName, "-", "_")
}

func (p Project) ValidateProjectName() error {
	if p.ProjectName == "" {
		return fmt.Errorf("project name cannot be empty")
	}
	if !projectNameRegex.MatchString(p.ProjectName) {
		return fmt.Errorf("'%s' may only contain letters, numbers, hyphens, and underscores, and must start with a letter", p.ProjectName)
	}
	if err := ValidatePythonIdentifier(p.ModuleName()); err != nil {
		return err
	}
	return validateNoModuleShadowing(p.ProjectName, p.ModuleName())
}

func (p Project) ValidateAppName() error {
	if p.AppName == "" {
		return nil
	}
	if err := ValidatePythonIdentifier(p.AppName); err != nil {
		return err
	}
	return validateNoModuleShadowing(p.AppName, p.AppName)
}

func (p *Project) ApplyFeatures(selected []Feature) {
	p.CreateTemplates = false
	p.CreateAppTemplates = false
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
			p.CreateAppTemplates = true
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
	return p.CreateAppTemplates || p.HasAPI() || p.SetupHTMX || p.SetupAllAuth
}
