package tui

import (
	"errors"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/vaibdix/django-stack/internal/config"
	"github.com/vaibdix/django-stack/internal/report"
)

func TestEventReporterPublishesToTheChannel(t *testing.T) {
	t.Parallel()

	events := make(chan tea.Msg, 4)
	var rep report.Reporter = eventReporter{events: events, done: make(chan struct{})}

	rep.Step("created venv")
	rep.Started("install django", "installing django")
	rep.Finished("install django", "installing django", time.Second, nil)

	if got, ok := (<-events).(scaffoldStepMsg); !ok || string(got) != "created venv" {
		t.Errorf("first event = %#v, want scaffoldStepMsg(%q)", got, "created venv")
	}
	if got, ok := (<-events).(scaffoldProgressMsg); !ok || string(got) != "installing django" {
		t.Errorf("second event = %#v, want scaffoldProgressMsg(%q)", got, "installing django")
	}
	if got, ok := (<-events).(scaffoldFinishedMsg); !ok {
		t.Errorf("third event = %#v, want scaffoldFinishedMsg", got)
	}
}

func TestUpdateAppliesScaffoldStepOnTheMainGoroutine(t *testing.T) {
	t.Parallel()

	m := newModelForViewTest()
	m.step = stepSetup
	m.scaffoldEvents = make(chan tea.Msg, 1)

	updated, cmd := m.Update(scaffoldStepMsg("created venv"))
	got := mustModel(updated)

	if len(got.stepMessages) != 1 || got.stepMessages[0] != "created venv" {
		t.Errorf("StepMessages = %q, want [created venv]", got.stepMessages)
	}
	if cmd == nil {
		t.Error("Update returned no command; the event stream would stall")
	}
}

func TestTheBarAdvancesWhenAStepFinishesNotWhenItStarts(t *testing.T) {
	t.Parallel()

	m := newModelForViewTest()
	m.step = stepSetup
	m.totalSteps = 4
	m.scaffoldEvents = make(chan tea.Msg, 1)

	updated, _ := m.Update(scaffoldProgressMsg("installing django"))
	started := mustModel(updated)
	if started.completedSteps != 0 || started.progressStatus != "installing django" {
		t.Errorf("after the start: CompletedSteps = %d, ProgressStatus = %q; want 0 and the step's label",
			started.completedSteps, started.progressStatus)
	}
	updated, _ = started.Update(scaffoldFinishedMsg{})
	finished := mustModel(updated)
	if finished.completedSteps != 1 {
		t.Errorf("after the finish: CompletedSteps = %d, want 1", finished.completedSteps)
	}
}

func TestUpdateRecordsGenerationFailure(t *testing.T) {
	t.Parallel()

	m := newModelForViewTest()
	m.step = stepSetup

	updated, _ := m.Update(scaffoldDoneMsg{Err: errFailedScaffold})
	got := mustModel(updated)

	if got.failure == nil {
		t.Fatal("Update did not record the generation error")
	}
	if !strings.Contains(got.progressStatus, "Error") {
		t.Errorf("ProgressStatus = %q, want it to report the failure", got.progressStatus)
	}
}

func TestCancelGenerationIsSafeWhenIdle(t *testing.T) {
	t.Parallel()

	m := &Model{}
	m.cancelGeneration()
}

func TestStartGenerationStreamsEventsWithoutTouchingTheModel(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		m := newModelForViewTest()
		m.choices = config.Choices{Name: "", Database: config.SQLite}
		m.step = stepSetup
		m.totalSteps = 3
		m.progressStatus = "untouched"

		cmd := m.startGeneration()
		if cmd == nil {
			t.Fatal("StartGeneration returned no command")
		}

		synctest.Wait()

		if len(m.stepMessages) != 0 {
			t.Errorf("the generator appended %d step messages to the model directly: %q",
				len(m.stepMessages), m.stepMessages)
		}
		if m.progressStatus != "untouched" {
			t.Errorf("ProgressStatus = %q, want it unchanged: only Update may write it", m.progressStatus)
		}
		if m.completedSteps != 0 {
			t.Errorf("CompletedSteps = %d, want 0: progress is counted in Update, not by the generator", m.completedSteps)
		}
		if m.failure != nil {
			t.Errorf("Error = %v, want nil: the failure travels as a message, not as a field write", m.failure)
		}

		var sawDone bool
		for msg := cmd(); msg != nil; {
			if _, done := msg.(scaffoldDoneMsg); done {
				sawDone = true
			}
			updated, next := m.Update(msg)
			m = mustModel(updated)
			if next == nil {
				break
			}
			cmd = next
			msg = cmd()
		}

		if !sawDone {
			t.Error("the stream ended without a scaffoldDoneMsg; the setup screen would wait forever")
		}
		if m.failure == nil {
			t.Error("an empty project name must surface as an error once Update has applied the message")
		}
	})
}

var errFailedScaffold = errScaffold("scaffold failed")

type errScaffold string

func (e errScaffold) Error() string { return string(e) }

func TestCtrlCDuringARunWaitsForTheCleanupBeforeQuitting(t *testing.T) {
	t.Parallel()

	m := questionModel(t, stepSetup)
	finished := make(chan struct{})
	cancelled := false
	m.generation = &generationRun{done: finished}
	m.cancelScaffold = func() { cancelled = true }

	_, cmd := m.Update(ctrlCKey)
	if !cancelled {
		t.Fatal("ctrl+c did not cancel the running scaffold")
	}
	if !m.cancelling || cmd == nil {
		t.Fatalf("ctrl+c quit at once instead of waiting for the rollback (cancelling=%v)", m.cancelling)
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
	t.Parallel()

	m := questionModel(t, stepSetup)
	m.generation = &generationRun{done: make(chan struct{})}
	m.cancelScaffold = func() {}

	m.Update(ctrlCKey)
	if !quits(m, ctrlCKey) {
		t.Error("a second ctrl+c did not force the quit")
	}
}

func TestTheReporterNeverBlocksOnceTheRunIsCancelled(t *testing.T) {
	t.Parallel()

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

func TestErrReportsHowTheRunEnded(t *testing.T) {
	t.Parallel()

	if err := (&Model{}).Err(); err != nil {
		t.Errorf("a model that never generated reports %v, want nil", err)
	}

	done := make(chan struct{})
	m := &Model{generation: &generationRun{done: done}, generationRoot: "/tmp/demo"}
	if err := m.Err(); err == nil || !strings.Contains(err.Error(), "/tmp/demo") {
		t.Errorf("a run still going at exit reports %v, want an error naming what to clean up", err)
	}

	m.generation.err = errFailedScaffold
	close(done)
	if err := m.Err(); !errors.Is(err, errFailedScaffold) {
		t.Errorf("a finished run reports %v, want the generation error", err)
	}
}

func TestCancellingWaitsForARollbackThatOutlastsAnyShutdownDeadline(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		m := &Model{generation: &generationRun{done: make(chan struct{})}, cancelScaffold: func() {}}
		quit := make(chan tea.Msg, 1)
		cmd := m.cancelAndWait()
		go func() { quit <- cmd() }()

		time.Sleep(3 * time.Minute)
		synctest.Wait()
		select {
		case msg := <-quit:
			t.Fatalf("the program quit (%T) while the rollback was still running", msg)
		default:
		}

		close(m.generation.done)
		if _, ok := (<-quit).(tea.QuitMsg); !ok {
			t.Error("the program did not quit once the rollback finished")
		}
	})
}
