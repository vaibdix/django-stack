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

type Model struct {
	config.Project

	// Clock is today's date for the Django support table. NewModel sets it to
	// time.Now; tests pin it so a series reaching end of life cannot break them.
	Clock func() time.Time

	Step                   Step
	AvailablePythonOptions []pyenv.Option
	AvailableDjangoOptions []DjangoOption
	DjangoVersionNote      string
	djangoRequest          int
	PythonDiscoveryNote    string
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
	StackForm              *huh.Form
	SelectedStack          config.Stack
	DockerForm             *huh.Form
	DockerRemedy           DockerRemedy
	DockerProblem          string
	DockerChecking         bool
	SelectedOptions        []config.Feature
	StepMessages           []string
	SplashCountdown        int

	Width  int
	Height int

	TotalSteps     int
	CompletedSteps int
	BlinkTimer     time.Time
	EyesOpen       bool

	// NodeManager is the package manager the Tailwind CSS watch runs with; it
	// is set only for a Tailwind stack.
	NodeManager scaffold.NodeManager

	Runner command.Runner

	scaffoldEvents chan tea.Msg

	generation *generationRun

	generationRoot string

	Cancelling bool

	cancelScaffold context.CancelFunc
}

func (m *Model) runner() command.Runner {
	if m.Runner == nil {
		return command.NewOS()
	}
	return m.Runner
}

func (m *Model) SetProgram(p *tea.Program) {
	m.Program = p
}

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

func (m *Model) RecentStepMessages(n int) []string {
	if n <= 0 {
		return nil
	}

	start := max(len(m.StepMessages)-n, 0)

	recent := make([]string, len(m.StepMessages)-start)
	copy(recent, m.StepMessages[start:])
	return recent
}

func (m *Model) GetActiveForm() *huh.Form {
	// While Docker is being probed the check screen shows progress, not its
	// remedy form.
	if m.Step == StepDockerCheck && m.DockerChecking {
		return nil
	}
	if slot := steps[m.Step].form; slot != nil {
		return *slot(m)
	}
	return nil
}

func (m *Model) watchCSSCommand() string {
	return m.NodeManager.RunCommand("watch:css")
}
