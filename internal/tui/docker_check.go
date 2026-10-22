package tui

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"

	"github.com/vaibdix/django-stack/internal/config"
	"github.com/vaibdix/django-stack/internal/dbdocker"
)

type dockerRemedy string

const (
	dockerRetry      dockerRemedy = "retry"
	dockerUseSQLite  dockerRemedy = "sqlite"
	dockerChangeStep dockerRemedy = "change"
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
	runner := m.commandRunner()
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
	if m.step != stepDockerCheck {
		return nil
	}

	m.dockerChecking = false
	m.dockerProblem = msg.Problem

	if msg.Ready {
		m.step = stepStack
		return []tea.Cmd{m.stackForm.Init()}
	}

	m.dockerRemedy = dockerRetry
	m.dockerForm = buildDockerRemedyForm(&m.dockerRemedy, m.project.DatabaseType)
	m.syncFormWidths()
	return []tea.Cmd{m.dockerForm.Init()}
}

func buildDockerRemedyForm(target *dockerRemedy, database config.Database) *huh.Form {
	return newForm(
		huh.NewGroup(
			huh.NewSelect[dockerRemedy]().
				Title("Docker is not ready").
				Description("A "+database.Spec().Label+" container cannot be started until Docker is running.").
				Options(
					huh.NewOption("I started Docker - check again", dockerRetry),
					huh.NewOption("Use SQLite instead", dockerUseSQLite),
					huh.NewOption("Go back and choose a different database", dockerChangeStep),
				).
				Value(target),
		),
	)
}

func (m *Model) beginDockerCheck() []tea.Cmd {
	m.step = stepDockerCheck
	m.dockerChecking = true
	m.dockerProblem = ""
	return []tea.Cmd{m.spinner.Tick, m.checkDockerCmd()}
}

func (m *Model) applyDockerRemedy() []tea.Cmd {
	switch m.dockerRemedy {
	case dockerUseSQLite:
		m.project.DatabaseType = config.SQLite
		m.dockerProblem = ""
		m.step = stepStack
		return []tea.Cmd{m.stackForm.Init()}

	case dockerChangeStep:
		m.dockerProblem = ""
		return m.goToPreviousStep()

	default:
		return m.beginDockerCheck()
	}
}
