package tui

import (
	"strings"
	"testing"
	"testing/synctest"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/vaibdix/django-cli/internal/config"
	"github.com/vaibdix/django-cli/internal/scaffold"
)

func TestEventReporterPublishesToTheChannel(t *testing.T) {
	t.Parallel()

	events := make(chan tea.Msg, 4)
	var rep scaffold.Reporter = eventReporter{events: events, done: make(chan struct{})}

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
	synctest.Test(t, func(t *testing.T) {
		m := newModelForViewTest()
		m.Project = config.Project{ProjectName: "", DatabaseType: config.SQLite}
		m.Step = StepSetup
		m.TotalSteps = 3
		m.ProgressStatus = "untouched"

		cmd := m.StartGeneration()
		if cmd == nil {
			t.Fatal("StartGeneration returned no command")
		}

		synctest.Wait()

		if len(m.StepMessages) != 0 {
			t.Errorf("the generator appended %d step messages to the model directly: %q",
				len(m.StepMessages), m.StepMessages)
		}
		if m.ProgressStatus != "untouched" {
			t.Errorf("ProgressStatus = %q, want it unchanged: only Update may write it", m.ProgressStatus)
		}
		if m.CompletedSteps != 0 {
			t.Errorf("CompletedSteps = %d, want 0: progress is counted in Update, not by the generator", m.CompletedSteps)
		}
		if m.Error != nil {
			t.Errorf("Error = %v, want nil: the failure travels as a message, not as a field write", m.Error)
		}

		var sawDone bool
		for msg := cmd(); msg != nil; {
			if _, done := msg.(scaffoldDoneMsg); done {
				sawDone = true
			}
			updated, next := m.Update(msg)
			m = updated.(*Model)
			if next == nil {
				break
			}
			cmd = next
			msg = cmd()
		}

		if !sawDone {
			t.Error("the stream ended without a scaffoldDoneMsg; the setup screen would wait forever")
		}
		if m.Error == nil {
			t.Error("an empty project name must surface as an error once Update has applied the message")
		}
	})
}

var errFailedScaffold = errScaffold("scaffold failed")

type errScaffold string

func (e errScaffold) Error() string { return string(e) }

func TestCtrlCDuringARunWaitsForTheCleanupBeforeQuitting(t *testing.T) {
	m := questionModel(t, StepSetup)
	finished := make(chan struct{})
	cancelled := false
	m.scaffoldFinished = finished
	m.cancelScaffold = func() { cancelled = true }

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if !cancelled {
		t.Fatal("ctrl+c did not cancel the running scaffold")
	}
	if !m.Cancelling || cmd == nil {
		t.Fatalf("ctrl+c quit at once instead of waiting for the rollback (cancelling=%v)", m.Cancelling)
	}

	result := make(chan tea.Msg, 1)
	go func() { result <- cmd() }()
	select {
	case msg := <-result:
		t.Fatalf("the program quit (%T) while the scaffold was still cleaning up", msg)
	case <-time.After(100 * time.Millisecond):
	}

	close(finished)
	select {
	case msg := <-result:
		if _, ok := msg.(tea.QuitMsg); !ok {
			t.Errorf("after the scaffold stopped, got %T, want tea.QuitMsg", msg)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("the program never quit after the scaffold stopped")
	}
}

func TestASecondCtrlCForcesTheQuit(t *testing.T) {
	m := questionModel(t, StepSetup)
	m.scaffoldFinished = make(chan struct{})
	m.cancelScaffold = func() {}

	m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if !quits(m, tea.KeyMsg{Type: tea.KeyCtrlC}) {
		t.Error("a second ctrl+c did not force the quit")
	}
}

func TestTheReporterNeverBlocksOnceTheRunIsCancelled(t *testing.T) {
	done := make(chan struct{})
	close(done)
	rep := eventReporter{events: make(chan tea.Msg), done: done}

	returned := make(chan struct{})
	go func() {
		rep.Step("a step nobody reads")
		close(returned)
	}()
	select {
	case <-returned:
	case <-time.After(time.Second):
		t.Fatal("a cancelled run blocked on reporting, so it could never finish its rollback")
	}
}
