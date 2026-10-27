package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/vaibdix/django-stack/internal/command/commandtest"
	"github.com/vaibdix/django-stack/internal/config"
)

func dockerModel(t *testing.T, database config.Database, fake *commandtest.Fake) *Model {
	t.Helper()

	m := questionModel(t, stepDatabase)
	m.choices.Database = database
	m.runner = fake
	m.rebuildForm(stepDatabase).Init()
	return m
}

func TestPickingADockerDatabaseChecksDockerBeforeAnythingIsWritten(t *testing.T) {
	t.Parallel()

	fake := commandtest.NewFake()
	fake.FailWith("docker info", "cannot connect to the Docker daemon")

	m := settle(dockerModel(t, config.PostgresDocker, fake), nil)
	m = advance(m, enterKey)

	if m.step != stepDockerCheck {
		t.Fatalf("selecting a Docker database advanced to %v; it must stop at the Docker check", m.step)
	}
	if m.dockerProblem == "" {
		t.Fatal("the Docker check reported no problem even though `docker info` failed")
	}
	if !strings.Contains(m.dockerProblem, "not running") {
		t.Errorf("problem = %q, want it to say Docker is not running", m.dockerProblem)
	}
}

func TestMissingDockerIsReportedDifferentlyFromStoppedDocker(t *testing.T) {
	t.Parallel()

	fake := commandtest.NewFake()
	fake.FailWith("docker --version", "command not found")

	m := settle(dockerModel(t, config.MySQLDocker, fake), nil)
	m = advance(m, enterKey)

	if !strings.Contains(m.dockerProblem, "not installed") {
		t.Errorf("problem = %q, want it to say Docker is not installed", m.dockerProblem)
	}
}

func TestAWorkingDockerPassesStraightThroughToTheStackStep(t *testing.T) {
	t.Parallel()

	m := settle(dockerModel(t, config.PostgresDocker, commandtest.NewFake()), nil)
	m = advance(m, enterKey)

	if m.step != stepStack {
		t.Fatalf("with Docker running the check must not stop the user, got %v", m.step)
	}
	if m.dockerProblem != "" {
		t.Errorf("a passing check left a problem set: %q", m.dockerProblem)
	}
}

func TestSQLiteSkipsTheDockerCheckEntirely(t *testing.T) {
	t.Parallel()

	fake := commandtest.NewFake()
	fake.FailWith("docker --version", "command not found")

	m := settle(dockerModel(t, config.SQLite, fake), nil)
	m = advance(m, enterKey)

	if m.step != stepStack {
		t.Fatalf("SQLite needs no Docker, but the flow reached %v", m.step)
	}
	for _, call := range fake.CommandLines() {
		if strings.HasPrefix(call, "docker") {
			t.Errorf("SQLite triggered a Docker probe: %q", call)
		}
	}
}

func TestChoosingSQLiteAtTheDockerGateKeepsTheRunGoing(t *testing.T) {
	t.Parallel()

	fake := commandtest.NewFake()
	fake.FailWith("docker info", "cannot connect to the Docker daemon")

	m := settle(dockerModel(t, config.PostgresDocker, fake), nil)
	m = advance(m, enterKey)
	if m.step != stepDockerCheck {
		t.Fatalf("expected to be held at the Docker check, got %v", m.step)
	}

	m.dockerRemedy = dockerUseSQLite
	m = settle(m, tea.Batch(m.applyDockerRemedy()...))

	if m.choices.Database != config.SQLite {
		t.Errorf("database = %q, want the run to fall back to SQLite", m.choices.Database)
	}
	if m.step != stepStack {
		t.Errorf("after falling back to SQLite the flow should continue, got %v", m.step)
	}
}

func TestRetryingTheDockerGateChecksAgain(t *testing.T) {
	t.Parallel()

	fake := commandtest.NewFake()
	fake.FailWith("docker info", "cannot connect to the Docker daemon")

	m := settle(dockerModel(t, config.PostgresDocker, fake), nil)
	m = advance(m, enterKey)

	before := strings.Count(strings.Join(fake.CommandLines(), "\n"), "docker info")

	m.dockerRemedy = dockerRetry
	m = settle(m, tea.Batch(m.applyDockerRemedy()...))

	after := strings.Count(strings.Join(fake.CommandLines(), "\n"), "docker info")
	if after <= before {
		t.Fatalf("retry did not re-probe Docker (%d probes before, %d after)", before, after)
	}
	if m.step != stepDockerCheck {
		t.Errorf("Docker is still down, so the gate must hold; got %v", m.step)
	}
}

func TestLeavingTheDockerGateClearsItsState(t *testing.T) {
	t.Parallel()

	fake := commandtest.NewFake()
	fake.FailWith("docker info", "cannot connect to the Docker daemon")

	m := settle(dockerModel(t, config.PostgresDocker, fake), nil)
	m = advance(m, enterKey)

	m = press(m, leftKey)

	if m.step != stepDatabase {
		t.Fatalf("left from the Docker check landed on %v, want StepDatabase", m.step)
	}
	if m.dockerChecking || m.dockerProblem != "" {
		t.Errorf("the Docker check left state behind: checking=%v problem=%q", m.dockerChecking, m.dockerProblem)
	}
}
