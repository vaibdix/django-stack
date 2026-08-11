package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/vaibdix/django-cli/internal/config"
	"github.com/vaibdix/django-cli/internal/scaffold"
)

func TestEventReporterPublishesToTheChannel(t *testing.T) {
	t.Parallel()

	events := make(chan tea.Msg, 4)
	var rep scaffold.Reporter = eventReporter{events: events}

	rep.Step("created venv")
	rep.Progress("installing django")

	if got, ok := (<-events).(scaffoldStepMsg); !ok || string(got) != "created venv" {
		t.Errorf("first event = %#v, want scaffoldStepMsg(%q)", got, "created venv")
	}
	if got, ok := (<-events).(scaffoldProgressMsg); !ok || string(got) != "installing django" {
		t.Errorf("second event = %#v, want scaffoldProgressMsg(%q)", got, "installing django")
	}
}

func TestUpdateAppliesScaffoldStepOnTheMainGoroutine(t *testing.T) {
	t.Parallel()

	m := newModelForViewTest()
	m.Step = StepSetup
	m.scaffoldEvents = make(chan tea.Msg, 1)

	updated, cmd := m.Update(scaffoldStepMsg("created venv"))
	got := updated.(*Model)

	if len(got.StepMessages) != 1 || got.StepMessages[0] != "created venv" {
		t.Errorf("StepMessages = %q, want [created venv]", got.StepMessages)
	}
	if cmd == nil {
		t.Error("Update returned no command; the event stream would stall")
	}
}

func TestUpdateAdvancesProgressOnScaffoldProgress(t *testing.T) {
	t.Parallel()

	m := newModelForViewTest()
	m.Step = StepSetup
	m.TotalSteps = 4
	m.scaffoldEvents = make(chan tea.Msg, 1)

	updated, _ := m.Update(scaffoldProgressMsg("installing django"))
	got := updated.(*Model)

	if got.CompletedSteps != 1 {
		t.Errorf("CompletedSteps = %d, want 1", got.CompletedSteps)
	}
	if got.ProgressStatus != "installing django" {
		t.Errorf("ProgressStatus = %q, want %q", got.ProgressStatus, "installing django")
	}
}

func TestUpdateRecordsGenerationFailure(t *testing.T) {
	t.Parallel()

	m := newModelForViewTest()
	m.Step = StepSetup

	updated, _ := m.Update(scaffoldDoneMsg{Err: errFailedScaffold})
	got := updated.(*Model)

	if got.Error == nil {
		t.Fatal("Update did not record the generation error")
	}
	if !strings.Contains(got.ProgressStatus, "Error") {
		t.Errorf("ProgressStatus = %q, want it to report the failure", got.ProgressStatus)
	}
}

func TestCancelGenerationIsSafeWhenIdle(t *testing.T) {
	t.Parallel()

	m := &Model{}
	m.CancelGeneration()
}

func TestStartGenerationStreamsEventsWithoutTouchingTheModel(t *testing.T) {
	t.Parallel()

	m := newModelForViewTest()
	m.Project = config.Project{ProjectName: "", DatabaseType: config.SQLite}
	m.Step = StepSetup
	m.TotalSteps = 3
	cmd := m.StartGeneration()
	if cmd == nil {
		t.Fatal("StartGeneration returned no command")
	}

	for i := 0; i < 20; i++ {
		msg := cmd()
		if msg == nil {
			t.Fatal("event stream closed before reporting completion")
		}
		updated, next := m.Update(msg)
		m = updated.(*Model)
		if _, done := msg.(scaffoldDoneMsg); done {
			return
		}
		if next == nil {
			t.Fatal("event stream stalled: Update returned no follow-up command")
		}
		cmd = next
	}
	t.Fatal("generation never reported completion")
}

var errFailedScaffold = errScaffold("scaffold failed")

type errScaffold string

func (e errScaffold) Error() string { return string(e) }
