package ui

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
)

var pythonIdentifierRegex = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
var projectNameRegex = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]*$`)
var pythonReservedWords = map[string]bool{
	"False": true, "None": true, "True": true, "and": true, "as": true,
	"assert": true, "async": true, "await": true, "break": true, "class": true,
	"continue": true, "def": true, "del": true, "elif": true, "else": true,
	"except": true, "finally": true, "for": true, "from": true, "global": true,
	"if": true, "import": true, "in": true, "is": true, "lambda": true,
	"nonlocal": true, "not": true, "or": true, "pass": true, "raise": true,
	"return": true, "try": true, "while": true, "with": true, "yield": true,
}

const defaultCommandTimeout = 10 * time.Minute

func validatePythonIdentifier(name string) error {
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

func (m *Model) ValidateProjectName() error {
	if m.ProjectName == "" {
		return fmt.Errorf("project name cannot be empty")
	}
	if !projectNameRegex.MatchString(m.ProjectName) {
		return fmt.Errorf("'%s' may only contain letters, numbers, hyphens, and underscores, and must start with a letter", m.ProjectName)
	}
	return validatePythonIdentifier(m.ProjectModuleName())
}

func (m *Model) ValidateAppName() error {
	if m.AppName == "" {
		return nil
	}
	return validatePythonIdentifier(m.AppName)
}

// ProjectConfig holds all project-level configuration selected by the user.
// It is embedded in Model so that field access remains unchanged.
// Generator functions can accept *ProjectConfig instead of *Model to
// avoid coupling to UI state.
type ProjectConfig struct {
	ProjectName            string
	DjangoVersion          string
	SelectedPythonPath     string
	SelectedPythonVersion  string
	AppName                string
	DatabaseType           string
	PostgresDockerPort     string
	PostgresDockerPassword string
	MySQLDockerPort        string
	MySQLDockerPassword    string
	Features               []string
	CreateTemplates        bool
	CreateAppTemplates     bool
	RunServer              bool
	InitializeGit          bool
	SetupTailwind          bool
	SetupRestFramework     bool
	SetupHTMX              bool
	SetupAllAuth           bool
	SetupRuff              bool
	SetupMypy              bool
	SetupPreCommit         bool
	SetupMultiEnv          bool
	UseGlobalTemplates     bool
	StartDevServer         bool
}

// Model holds the full TUI state: project configuration (embedded),
// UI widgets, progress tracking, and form state.
type Model struct {
	ProjectConfig

	Step                   Step
	AvailablePythonOptions []PythonOption
	AvailableDjangoOptions []DjangoOption
	DjangoVersionNote      string
	Spinner                spinner.Model
	Progress               progress.Model
	ProgressStatus         string
	Error                  error
	Done                   bool
	Program                *tea.Program
	ProjectNameForm        *huh.Form
	PythonForm             *huh.Form
	DjangoVersionForm      *huh.Form
	DatabaseForm           *huh.Form
	ProjectConfigForm      *huh.Form
	AppNameForm            *huh.Form
	DevServerForm          *huh.Form
	SelectedOptions        []string
	StepMessages           []string
	SplashCountdown        int
	Width                  int
	TotalSteps             int
	CompletedSteps         int
	BlinkTimer             time.Time
	EyesOpen               bool
	progressMu             sync.Mutex
	StepMessagesMu         sync.Mutex
}

const (
	StepCountVenv      = 1
	StepCountDjango    = 1
	StepCountTemplates = 1
	StepCountApp       = 1
	StepCountGit       = 1
	StepCountTailwind  = 1
	StepCountRest      = 1
	StepCountHTMX      = 1
	StepCountAllAuth   = 1
	StepCountRuff      = 1
	StepCountMypy      = 1
	StepCountPreCommit = 1
	StepCountMultiEnv  = 1
	StepCountLint      = 1
	StepCountFinal     = 1
)

func (m *Model) SetProgram(p *tea.Program) {
	m.Program = p
}

func (m *Model) CalculateTotalSteps() int {
	steps := StepCountVenv + StepCountDjango

	if m.CreateTemplates {
		steps += StepCountTemplates
	}
	if m.AppName != "" {
		steps += StepCountApp
	}
	if m.InitializeGit {
		steps += StepCountGit
	}
	if m.SetupTailwind {
		steps += StepCountTailwind
	}
	if m.SetupRestFramework {
		steps += StepCountRest
	}
	if m.SetupHTMX {
		steps += StepCountHTMX
	}
	if m.SetupAllAuth {
		steps += StepCountAllAuth
	}
	if m.SetupRuff {
		steps += StepCountRuff
	}
	if m.SetupMypy {
		steps += StepCountMypy
	}
	if m.SetupPreCommit {
		steps += StepCountPreCommit
	}
	if m.SetupMultiEnv {
		steps += StepCountMultiEnv
	}
	if m.SetupPreCommit || m.SetupRuff {
		steps += StepCountLint
	}
	steps += StepCountFinal

	return steps
}

func (m *Model) UpdateProgress(status string) {
	m.progressMu.Lock()
	m.CompletedSteps++
	completed := m.CompletedSteps
	total := m.TotalSteps
	m.progressMu.Unlock()

	prog := float64(completed) / float64(total)
	if m.Program != nil {
		m.Program.Send(ProjectProgressMsg{Percent: prog, Status: status})
	}
}

func (m *Model) AppendStepMessage(msg string) {
	m.StepMessagesMu.Lock()
	m.StepMessages = append(m.StepMessages, msg)
	m.StepMessagesMu.Unlock()
}

func (m *Model) GetActiveForm() *huh.Form {
	switch m.Step {
	case StepProjectName:
		return m.ProjectNameForm
	case StepPythonSelection:
		return m.PythonForm
	case StepDjangoVersion:
		return m.DjangoVersionForm
	case StepDatabase:
		return m.DatabaseForm
	case StepProjectConfig:
		return m.ProjectConfigForm
	case StepAppName:
		return m.AppNameForm
	case StepDevServerPrompt:
		return m.DevServerForm
	default:
		return nil
	}
}

func (m *Model) ProjectModuleName() string {
	return strings.ReplaceAll(m.ProjectName, "-", "_")
}
