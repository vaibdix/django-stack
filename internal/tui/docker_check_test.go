package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/vaibdix/django-stack/internal/command/commandtest"
	"github.com/vaibdix/django-stack/internal/config"
)

func dockerModel(t *testing.T, database config.Database, fake *commandtest.Fake) *Model {
	t.Helper()

	m := questionModel(t, StepDatabase)
	m.DatabaseType = database
	m.Runner = fake
	m.rebuildForm(StepDatabase).Init()
	return m
}

func TestPickingADockerDatabaseChecksDockerBeforeAnythingIsWritten(t *testing.T) {
	fake := commandtest.NewFake()
	fake.FailWith("docker info", "cannot connect to the Docker daemon")

	m := settle(dockerModel(t, config.PostgresDocker, fake), nil)
	m = advance(m, tea.KeyEnter)

	if m.Step != StepDockerCheck {
		t.Fatalf("selecting a Docker database advanced to %v; it must stop at the Docker check", m.Step)
	}
	if m.DockerProblem == "" {
		t.Fatal("the Docker check reported no problem even though `docker info` failed")
	}
	if !strings.Contains(m.DockerProblem, "not running") {
		t.Errorf("problem = %q, want it to say Docker is not running", m.DockerProblem)
	}
}

func TestMissingDockerIsReportedDifferentlyFromStoppedDocker(t *testing.T) {
	fake := commandtest.NewFake()
	fake.FailWith("docker --version", "command not found")

	m := settle(dockerModel(t, config.MySQLDocker, fake), nil)
	m = advance(m, tea.KeyEnter)

	if !strings.Contains(m.DockerProblem, "not installed") {
		t.Errorf("problem = %q, want it to say Docker is not installed", m.DockerProblem)
	}
}

func TestAWorkingDockerPassesStraightThroughToTheStackStep(t *testing.T) {
	m := settle(dockerModel(t, config.PostgresDocker, commandtest.NewFake()), nil)
	m = advance(m, tea.KeyEnter)

	if m.Step != StepStack {
		t.Fatalf("with Docker running the check must not stop the user, got %v", m.Step)
	}
	if m.DockerProblem != "" {
		t.Errorf("a passing check left a problem set: %q", m.DockerProblem)
	}
}

func TestSQLiteSkipsTheDockerCheckEntirely(t *testing.T) {
	fake := commandtest.NewFake()
	fake.FailWith("docker --version", "command not found")

	m := settle(dockerModel(t, config.SQLite, fake), nil)
	m = advance(m, tea.KeyEnter)

	if m.Step != StepStack {
		t.Fatalf("SQLite needs no Docker, but the flow reached %v", m.Step)
	}
	for _, call := range fake.CommandLines() {
		if strings.HasPrefix(call, "docker") {
			t.Errorf("SQLite triggered a Docker probe: %q", call)
		}
	}
}

func TestChoosingSQLiteAtTheDockerGateKeepsTheRunGoing(t *testing.T) {
	fake := commandtest.NewFake()
	fake.FailWith("docker info", "cannot connect to the Docker daemon")

	m := settle(dockerModel(t, config.PostgresDocker, fake), nil)
	m = advance(m, tea.KeyEnter)
	if m.Step != StepDockerCheck {
		t.Fatalf("expected to be held at the Docker check, got %v", m.Step)
	}

	m.DockerRemedy = DockerUseSQLite
	m = settle(m, tea.Batch(m.applyDockerRemedy()...))

	if m.DatabaseType != config.SQLite {
		t.Errorf("database = %q, want the run to fall back to SQLite", m.DatabaseType)
	}
	if m.Step != StepStack {
		t.Errorf("after falling back to SQLite the flow should continue, got %v", m.Step)
	}
}

func TestRetryingTheDockerGateChecksAgain(t *testing.T) {
	fake := commandtest.NewFake()
	fake.FailWith("docker info", "cannot connect to the Docker daemon")

	m := settle(dockerModel(t, config.PostgresDocker, fake), nil)
	m = advance(m, tea.KeyEnter)

	before := strings.Count(strings.Join(fake.CommandLines(), "\n"), "docker info")

	m.DockerRemedy = DockerRetry
	m = settle(m, tea.Batch(m.applyDockerRemedy()...))

	after := strings.Count(strings.Join(fake.CommandLines(), "\n"), "docker info")
	if after <= before {
		t.Fatalf("retry did not re-probe Docker (%d probes before, %d after)", before, after)
	}
	if m.Step != StepDockerCheck {
		t.Errorf("Docker is still down, so the gate must hold; got %v", m.Step)
	}
}

func TestLeavingTheDockerGateClearsItsState(t *testing.T) {
	fake := commandtest.NewFake()
	fake.FailWith("docker info", "cannot connect to the Docker daemon")

	m := settle(dockerModel(t, config.PostgresDocker, fake), nil)
	m = advance(m, tea.KeyEnter)

	m = press(m, tea.KeyLeft)

	if m.Step != StepDatabase {
		t.Fatalf("left from the Docker check landed on %v, want StepDatabase", m.Step)
	}
	if m.DockerChecking || m.DockerProblem != "" {
		t.Errorf("the Docker check left state behind: checking=%v problem=%q", m.DockerChecking, m.DockerProblem)
	}
}
