package tui

import (
	"context"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"

	"github.com/vaibdix/django-cli/internal/config"
	"github.com/vaibdix/django-cli/internal/scaffold"
	"github.com/vaibdix/django-cli/internal/scaffold/command"

	"github.com/vaibdix/django-cli/internal/scaffold/pyenv"
)

// Model holds the full TUI state: project configuration (embedded),
// UI widgets, progress tracking, and form state.
type Model struct {
	config.Project

	Step                   Step
	AvailablePythonOptions []pyenv.Option
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
	SelectedOptions        []config.Feature
	StepMessages           []string
	SplashCountdown        int

	Width  int
	Height int

	TotalSteps     int
	CompletedSteps int
	BlinkTimer     time.Time
	EyesOpen       bool

	Runner command.Runner

	scaffoldEvents chan tea.Msg

	cancelScaffold context.CancelFunc
}

func (m *Model) runner() command.Runner {
	if m.Runner == nil {
		return command.NewOS()
	}
	return m.Runner
}

// SetProgram records the running program. It must be called before Run.
func (m *Model) SetProgram(p *tea.Program) {
	m.Program = p
}

// CalculateTotalSteps returns how many progress steps the selected feature set
// will report, which is the denominator of the progress bar.
//
// It asks the engine rather than counting features itself. It used to sum
// fifteen named constants that were every one of them 1, and the engine emitted
// progress updates from wherever seemed appropriate, so the declared total and
// the actual number of updates disagreed in every configuration — enabling
// Tailwind fired two updates against a budget of one, and the bar reached 200%
// in the minimal case. Both numbers now come from the same step list.
func (m *Model) CalculateTotalSteps() int {
	return scaffold.StepCount(m.Project)
}

func (m *Model) advanceProgress() float64 {
	m.CompletedSteps++

	if m.TotalSteps <= 0 {
		return 0
	}
	if m.CompletedSteps > m.TotalSteps {
		m.CompletedSteps = m.TotalSteps
	}
	return float64(m.CompletedSteps) / float64(m.TotalSteps)
}

const recentStepMessageCount = 3

// RecentStepMessages returns at most the last n step messages.
//
// No locking is needed: StepMessages is written only by Update and read only by
// View, both on Bubble Tea's single goroutine. It still returns a copy so
// callers cannot alias the model's storage.
func (m *Model) RecentStepMessages(n int) []string {
	if n <= 0 {
		return nil
	}

	start := len(m.StepMessages) - n
	if start < 0 {
		start = 0
	}

	recent := make([]string, len(m.StepMessages)-start)
	copy(recent, m.StepMessages[start:])
	return recent
}

// GetActiveForm returns the form for the current step, or nil if the current
// step does not show one.
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
