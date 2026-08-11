package tui

import (
	"context"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/vaibdix/django-cli/internal/scaffold"
)

// This file is the seam between the TUI and the scaffolding engine.
//
// The engine runs on its own goroutine and never touches the model. It reports
// through a Reporter that only writes to a channel; the update loop drains that
// channel as Bubble Tea messages and applies them on the main goroutine. That
// is what makes the model single-writer — previously the generator called
// straight back into the model while View was reading it.

// scaffoldStepMsg is one step message emitted by the generator.
type scaffoldStepMsg string

// scaffoldProgressMsg is one progress advance emitted by the generator.
type scaffoldProgressMsg string

// scaffoldDoneMsg reports the generator finishing, successfully or not.
type scaffoldDoneMsg struct{ Err error }

// eventReporter implements scaffold.Reporter by publishing to a channel.
//
// Sends are blocking on purpose: if the UI cannot keep up, the generator waits
// rather than dropping the log. The buffer makes that effectively never happen.
type eventReporter struct{ events chan<- tea.Msg }

func (r eventReporter) Step(message string) { r.events <- scaffoldStepMsg(message) }

func (r eventReporter) Progress(status string) { r.events <- scaffoldProgressMsg(status) }

// scaffoldEventBuffer is deep enough that the generator never blocks in
// practice; a full scaffold emits on the order of fifty messages.
const scaffoldEventBuffer = 128

// projectRoot resolves the directory the project will be created in.
func (m *Model) projectRoot() (string, error) {
	if filepath.IsAbs(m.ProjectName) {
		return m.ProjectName, nil
	}
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return filepath.Join(wd, m.ProjectName), nil
}

// newScaffoldEnv builds the engine's environment from the model's state.
//
// Cfg is copied, not shared: once generation starts the engine reads its own
// snapshot, so nothing the user does in the TUI can change it mid-run.
func (m *Model) newScaffoldEnv(root string, reporter scaffold.Reporter) *scaffold.Env {
	return &scaffold.Env{
		Cfg:      m.Project,
		Root:     root,
		Runner:   m.runner(),
		Reporter: reporter,
	}
}

// StartGeneration launches the scaffolder and returns the command that waits
// for its first event.
//
// Each event handler in Update re-issues waitForScaffoldEvent, so the stream is
// drained one message at a time without any goroutine touching the model.
func (m *Model) StartGeneration() tea.Cmd {
	root, err := m.projectRoot()
	if err != nil {
		return func() tea.Msg { return scaffoldDoneMsg{Err: err} }
	}

	events := make(chan tea.Msg, scaffoldEventBuffer)
	m.scaffoldEvents = events

	ctx, cancel := context.WithCancel(context.Background())
	m.cancelScaffold = cancel

	env := m.newScaffoldEnv(root, eventReporter{events: events})

	go func() {
		defer close(events)
		events <- scaffoldDoneMsg{Err: env.Generate(ctx)}
	}()

	return m.waitForScaffoldEvent()
}

// waitForScaffoldEvent blocks until the generator publishes its next message.
func (m *Model) waitForScaffoldEvent() tea.Cmd {
	events := m.scaffoldEvents
	if events == nil {
		return nil
	}
	return func() tea.Msg {
		msg, ok := <-events
		if !ok {
			return nil
		}
		return msg
	}
}

// CancelGeneration stops the running scaffolder, if any. Cancelling the context
// propagates to every child process the run started.
func (m *Model) CancelGeneration() {
	if m.cancelScaffold != nil {
		m.cancelScaffold()
	}
}

// startDevelopmentEnvironment opens the generated project in the editor and
// starts its dev servers.
//
// This runs on the main goroutine after generation has finished, so it reports
// directly rather than through the event channel.
func (m *Model) startDevelopmentEnvironment() {
	root, err := m.projectRoot()
	if err != nil {
		m.StepMessages = append(m.StepMessages, "Could not resolve the project directory: "+err.Error())
		return
	}
	m.newScaffoldEnv(root, directReporter{m}).StartDevelopmentEnvironment()
}

// directReporter applies messages straight to the model. It is only safe on the
// main goroutine, which is why the generator does not use it.
type directReporter struct{ m *Model }

func (r directReporter) Step(message string) {
	r.m.StepMessages = append(r.m.StepMessages, message)
}

func (r directReporter) Progress(status string) {
	r.m.ProgressStatus = status
}
