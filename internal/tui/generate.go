package tui

import (
	"context"
	"os"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/vaibdix/django-cli/internal/scaffold"
)

type scaffoldStepMsg string

type scaffoldProgressMsg string

type scaffoldDoneMsg struct{ Err error }

type eventReporter struct {
	events chan<- tea.Msg
	done   <-chan struct{}
}

func (r eventReporter) send(msg tea.Msg) {
	select {
	case r.events <- msg:
	case <-r.done:
	}
}

func (r eventReporter) Step(message string) { r.send(scaffoldStepMsg(message)) }

func (r eventReporter) Progress(status string) { r.send(scaffoldProgressMsg(status)) }

const shutdownTimeout = 30 * time.Second

const scaffoldEventBuffer = 128

const keepFailedEnv = "DJANGO_STACK_KEEP_FAILED"

func (m *Model) projectRoot() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return filepath.Join(wd, m.ProjectName), nil
}

func (m *Model) newScaffoldEnv(root string, reporter scaffold.Reporter) *scaffold.Env {
	return &scaffold.Env{
		Cfg:           m.Project,
		Root:          root,
		Runner:        m.runner(),
		Reporter:      reporter,
		KeepOnFailure: os.Getenv(keepFailedEnv) != "",
	}
}

func (m *Model) StartGeneration() tea.Cmd {
	root, err := m.projectRoot()
	if err != nil {
		return func() tea.Msg { return scaffoldDoneMsg{Err: err} }
	}

	events := make(chan tea.Msg, scaffoldEventBuffer)
	m.scaffoldEvents = events

	ctx, cancel := context.WithCancel(context.Background())
	m.cancelScaffold = cancel

	finished := make(chan struct{})
	m.scaffoldFinished = finished

	reporter := eventReporter{events: events, done: ctx.Done()}
	env := m.newScaffoldEnv(root, reporter)

	go func() {
		defer close(finished)
		defer close(events)
		reporter.send(scaffoldDoneMsg{Err: env.Generate(ctx)})
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

func (m *Model) CancelGeneration() {
	if m.cancelScaffold != nil {
		m.cancelScaffold()
	}
}

func (m *Model) generationRunning() bool {
	if m.scaffoldFinished == nil {
		return false
	}
	select {
	case <-m.scaffoldFinished:
		return false
	default:
		return true
	}
}

func (m *Model) cancelAndWait() tea.Cmd {
	m.Cancelling = true
	m.ProgressStatus = "Cancelling: stopping child processes and removing the partial project..."
	m.CancelGeneration()

	finished := m.scaffoldFinished
	return func() tea.Msg {
		timer := time.NewTimer(shutdownTimeout)
		defer timer.Stop()
		select {
		case <-finished:
		case <-timer.C:
		}
		return tea.QuitMsg{}
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
