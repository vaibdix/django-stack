package tui

import (
	"context"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/vaibdix/django-cli/internal/scaffold"
)

type scaffoldStepMsg string

type scaffoldProgressMsg string

type scaffoldDoneMsg struct{ Err error }

type eventReporter struct{ events chan<- tea.Msg }

func (r eventReporter) Step(message string) { r.events <- scaffoldStepMsg(message) }

func (r eventReporter) Progress(status string) { r.events <- scaffoldProgressMsg(status) }

const scaffoldEventBuffer = 128

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

func (m *Model) startDevelopmentEnvironment() {
	root, err := m.projectRoot()
	if err != nil {
		m.StepMessages = append(m.StepMessages, "Could not resolve the project directory: "+err.Error())
		return
	}
	m.newScaffoldEnv(root, directReporter{m}).StartDevelopmentEnvironment()
}

type directReporter struct{ m *Model }

func (r directReporter) Step(message string) {
	r.m.StepMessages = append(r.m.StepMessages, message)
}

func (r directReporter) Progress(status string) {
	r.m.ProgressStatus = status
}
