package tui

import (
	"context"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"

	"github.com/vaibdix/django-stack/internal/command"
	"github.com/vaibdix/django-stack/internal/config"
	"github.com/vaibdix/django-stack/internal/scaffold"

	"github.com/vaibdix/django-stack/internal/pyenv"
	"github.com/vaibdix/django-stack/internal/releases"
)

type Model struct {
	project config.Project

	startDevServer bool

	clock func() time.Time

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
	selectedStack          config.Stack
	dockerForm             *huh.Form
	dockerRemedy           dockerRemedy
	dockerProblem          string
	dockerChecking         bool
	selectedOptions        []config.Feature
	stepMessages           []string
	splashCountdown        int

	width  int
	height int

	totalSteps     int
	completedSteps int

	nodeManager scaffold.NodeManager

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
	return scaffold.StepCount(m.project)
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
