package tui

import (
	"context"
	"time"

	"charm.land/bubbles/v2/progress"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"

	"github.com/vaibdix/django-stack/internal/command"
	"github.com/vaibdix/django-stack/internal/config"
	"github.com/vaibdix/django-stack/internal/pyenv"
	"github.com/vaibdix/django-stack/internal/releases"
	"github.com/vaibdix/django-stack/internal/render"
	"github.com/vaibdix/django-stack/internal/scaffold"
)

type Model struct {
	choices  config.Choices
	resolved config.Resolved
	spec     config.Spec

	startDevServer bool

	clock func() time.Time

	discoverPython func(context.Context, command.Runner) ([]pyenv.Option, error)

	probeHost func(context.Context, command.Runner) scaffold.Host

	host scaffold.Host

	djangoOptions func(context.Context, *releases.Catalog, pyenv.Option, time.Time) ([]releases.Option, string)

	step                   wizardStep
	availablePythonOptions []pyenv.Option
	availableDjangoOptions []releases.Option
	djangoVersionNote      string
	djangoRequest          int
	pythonDiscoveryNote    string
	spinner                spinner.Model
	progress               progress.Model
	progressStatus         string
	failure                error
	done                   bool
	program                *tea.Program
	projectNameForm        *huh.Form
	pythonForm             *huh.Form
	djangoVersionForm      *huh.Form
	databaseForm           *huh.Form
	projectConfigForm      *huh.Form
	appNameForm            *huh.Form
	devServerForm          *huh.Form
	stackForm              *huh.Form
	dockerForm             *huh.Form
	dockerRemedy           dockerRemedy
	dockerProblem          string
	dockerChecking         bool
	selectedOptions        []config.AddOn
	stepMessages           []string
	splashCountdown        int

	width  int
	height int

	totalSteps     int
	completedSteps int

	nodeManager render.NodeManager

	runner command.Runner

	catalog *releases.Catalog

	engine scaffold.Options

	scaffoldEvents chan tea.Msg

	generation *generationRun

	generationRoot string

	cancelling bool

	cancelScaffold context.CancelFunc
}

func (m *Model) commandRunner() command.Runner {
	if m.runner == nil {
		return command.NewOS()
	}
	return m.runner
}

func (m *Model) SetProgram(p *tea.Program) {
	m.program = p
}

func (m *Model) calculateTotalSteps() int {
	spec, _, err := config.Parse(m.choices, m.interpreter(), time.Time{})
	if err != nil {
		return 0
	}
	return scaffold.StepCount(spec)
}

func (m *Model) advanceProgress() float64 {
	m.completedSteps++

	if m.totalSteps <= 0 {
		return 0
	}
	if m.completedSteps > m.totalSteps {
		m.completedSteps = m.totalSteps
	}
	return float64(m.completedSteps) / float64(m.totalSteps)
}

const recentStepMessageCount = 3

func (m *Model) recentStepMessages(n int) []string {
	if n <= 0 {
		return nil
	}

	start := max(len(m.stepMessages)-n, 0)

	recent := make([]string, len(m.stepMessages)-start)
	copy(recent, m.stepMessages[start:])
	return recent
}

func (m *Model) currentForm() *huh.Form {
	if m.step == stepDockerCheck && m.dockerChecking {
		return nil
	}
	if slot := steps()[m.step].form; slot != nil {
		return *slot(m)
	}
	return nil
}

func (m *Model) watchCSSCommand() string {
	return m.nodeManager.RunCommand("watch:css")
}
