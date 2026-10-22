package tui

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"

	"github.com/vaibdix/django-stack/internal/config"
	"github.com/vaibdix/django-stack/internal/scaffold/dbdocker"
)

type DockerRemedy string

const (
	DockerRetry      DockerRemedy = "retry"
	DockerUseSQLite  DockerRemedy = "sqlite"
	DockerChangeStep DockerRemedy = "change"
)

type dockerStatusMsg struct {
	Ready   bool
	Problem string
}

const (
	dockerMissingProblem = "Docker is not installed. Install Docker Desktop or OrbStack, then retry."
	dockerStoppedProblem = "Docker is installed but not running. Start Docker Desktop or OrbStack, then retry."
	dockerComposeProblem = "Docker Compose v2 is not available. Update Docker Desktop or OrbStack, then retry."
)

const dockerCheckTimeout = 15 * time.Second

func (m *Model) checkDockerCmd() tea.Cmd {
	runner := m.runner()
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), dockerCheckTimeout)
		defer cancel()

		if !dbdocker.Available(ctx, runner) {
			return dockerStatusMsg{Problem: dockerMissingProblem}
		}
		if !dbdocker.ComposeAvailable(ctx, runner) {
			return dockerStatusMsg{Problem: dockerComposeProblem}
		}
		if !dbdocker.Running(ctx, runner) {
			return dockerStatusMsg{Problem: dockerStoppedProblem}
		}
		return dockerStatusMsg{Ready: true}
	}
}

func (m *Model) applyDockerStatus(msg dockerStatusMsg) []tea.Cmd {
	if m.Step != StepDockerCheck {
		return nil
	}

	m.DockerChecking = false
	m.DockerProblem = msg.Problem

	if msg.Ready {
		m.Step = StepStack
		return []tea.Cmd{m.StackForm.Init()}
	}

	m.DockerRemedy = DockerRetry
	m.DockerForm = buildDockerRemedyForm(&m.DockerRemedy, m.DatabaseType)
	m.syncFormWidths()
	return []tea.Cmd{m.DockerForm.Init()}
}

func buildDockerRemedyForm(target *DockerRemedy, database config.Database) *huh.Form {
	engine := "PostgreSQL"
	if database == config.MySQLDocker {
		engine = "MySQL"
	}

	return newForm(
		huh.NewGroup(
			huh.NewSelect[DockerRemedy]().
				Title("Docker is not ready").
				Description("A "+engine+" container cannot be started until Docker is running.").
				Options(
					huh.NewOption("I started Docker - check again", DockerRetry),
					huh.NewOption("Use SQLite instead", DockerUseSQLite),
					huh.NewOption("Go back and choose a different database", DockerChangeStep),
				).
				Value(target),
		),
	)
}

func (m *Model) beginDockerCheck() []tea.Cmd {
	m.Step = StepDockerCheck
	m.DockerChecking = true
	m.DockerProblem = ""
	return []tea.Cmd{m.Spinner.Tick, m.checkDockerCmd()}
}

func (m *Model) applyDockerRemedy() []tea.Cmd {
	switch m.DockerRemedy {
	case DockerUseSQLite:
		m.DatabaseType = config.SQLite
		m.DockerProblem = ""
		m.Step = StepStack
		return []tea.Cmd{m.StackForm.Init()}

	case DockerChangeStep:
		m.DockerProblem = ""
		return m.goToPreviousStep()

	default:
		return m.beginDockerCheck()
	}
}
