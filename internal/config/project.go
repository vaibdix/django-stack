// Package config holds the project configuration a user assembles during setup,
// together with the rules for validating it.
//
// It has no dependencies beyond the standard library — in particular it knows
// nothing about Bubble Tea — so scaffolding logic that accepts a Project can be
// tested without a terminal.
package config

import (
	"fmt"
	"regexp"
	"strings"
)

// Database identifies the database backend to scaffold.
type Database string

const (
	// SQLite uses Django's default file-backed database; no container needed.
	SQLite Database = "sqlite"
	// PostgresDocker runs PostgreSQL in a container managed by this tool.
	PostgresDocker Database = "postgres-docker"
	// PostgresString targets an externally managed PostgreSQL instance.
	PostgresString Database = "postgres-string"
	// MySQLDocker runs MySQL in a container managed by this tool.
	MySQLDocker Database = "mysql-docker"
)

// UsesDocker reports whether the backend is a container this tool starts.
func (d Database) UsesDocker() bool {
	return d == PostgresDocker || d == MySQLDocker
}

// Feature is an optional capability the user can enable during setup.
//
// These values are the identities the scaffolder switches on. They are
// deliberately separate from the labels shown in the TUI, so reworded menu text
// cannot silently disable a feature.
type Feature string

// The features the setup screen offers, in menu order.
const (
	FeatureStandardProject Feature = "Standard Django Project"
	FeatureGit             Feature = "Initialize Git"
	FeatureTailwind        Feature = "Tailwind"
	FeatureRestFramework   Feature = "REST Framework"
	FeatureHTMX            Feature = "HTMX"
	FeatureAllAuth         Feature = "AllAuth"
	FeatureRuff            Feature = "Ruff"
	FeatureMypy            Feature = "Mypy"
	FeaturePreCommit       Feature = "Pre-commit"
	FeatureMultiEnv        Feature = "Multi-env"
)

// Project is everything the user chose about the project to be generated.
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
	RunServer          bool
	InitializeGit      bool
	SetupTailwind      bool
	SetupRestFramework bool
	SetupHTMX          bool
	SetupAllAuth       bool
	SetupRuff          bool
	SetupMypy          bool
	SetupPreCommit     bool
	SetupMultiEnv      bool
	UseGlobalTemplates bool
	StartDevServer     bool
}

var (
	// pythonIdentifierRegex matches a legal Python identifier.
	pythonIdentifierRegex = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

	// projectNameRegex is deliberately looser than pythonIdentifierRegex:
	// directory names may contain hyphens. ModuleName bridges the two.
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

// ValidatePythonIdentifier reports whether name may be used as a Python module,
// package or app name.
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

// ModuleName is the Python package name derived from the project name.
// Hyphens are legal in a directory name but not in a Python import, so they
// become underscores.
func (p Project) ModuleName() string {
	return strings.ReplaceAll(p.ProjectName, "-", "_")
}

// ValidateProjectName checks the project name both as a directory name and as
// the Python module it will be imported as.
func (p Project) ValidateProjectName() error {
	if p.ProjectName == "" {
		return fmt.Errorf("project name cannot be empty")
	}
	if !projectNameRegex.MatchString(p.ProjectName) {
		return fmt.Errorf("'%s' may only contain letters, numbers, hyphens, and underscores, and must start with a letter", p.ProjectName)
	}
	return ValidatePythonIdentifier(p.ModuleName())
}

// ValidateAppName checks the optional Django app name. An empty name is valid:
// the app is optional.
func (p Project) ValidateAppName() error {
	if p.AppName == "" {
		return nil
	}
	return ValidatePythonIdentifier(p.AppName)
}

// ApplyFeatures resets every optional feature flag and re-enables the ones in
// selected. It is called each time the user leaves the feature screen, so it
// must clear stale choices rather than accumulate them.
func (p *Project) ApplyFeatures(selected []Feature) {
	p.CreateTemplates = false
	p.CreateAppTemplates = false
	p.InitializeGit = false
	p.SetupTailwind = false
	p.SetupRestFramework = false
	p.SetupHTMX = false
	p.SetupAllAuth = false
	p.SetupRuff = false
	p.SetupMypy = false
	p.SetupPreCommit = false
	p.SetupMultiEnv = false

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
		}
	}

	// Pre-commit hooks live in a git repository, so selecting them implies git.
	if p.SetupPreCommit && !p.InitializeGit {
		p.InitializeGit = true
	}
}

// NeedsAppName reports whether the selected features require a Django app,
// which is what decides if the app-name screen is shown.
func (p Project) NeedsAppName() bool {
	return p.CreateAppTemplates || p.SetupRestFramework || p.SetupHTMX || p.SetupAllAuth
}
