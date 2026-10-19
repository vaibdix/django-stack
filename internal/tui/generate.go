package tui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

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

const scaffoldEventBuffer = 128

// generationRun is the one background call to scaffold.Env.Generate. The
// goroutine writes err before it closes done, and err is read only after done
// is closed, so the close orders the write before every read.
type generationRun struct {
	done chan struct{}
	err  error
}

func (g *generationRun) finished() bool {
	select {
	case <-g.done:
		return true
	default:
		return false
	}
}

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
		KeepOnFailure: os.Getenv(scaffold.KeepFailedEnv) != "",
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

	run := &generationRun{done: make(chan struct{})}
	m.generation = run
	m.generationRoot = root

	reporter := eventReporter{events: events, done: ctx.Done()}
	env := m.newScaffoldEnv(root, reporter)

	go func() {
		defer close(run.done)
		defer close(events)
		run.err = env.Generate(ctx)
		reporter.send(scaffoldDoneMsg{Err: run.err})
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
	return m.generation != nil && !m.generation.finished()
}

// Err is how the run ended, for the process exit status once the program has
// quit. Quitting before generation starts is not an error. A run that was
// still going when the program quit (a forced second ctrl+c) is one, because
// its rollback never finished and the partial project may still be on disk.
func (m *Model) Err() error {
	switch {
	case m.generation == nil:
		return nil
	case !m.generation.finished():
		return fmt.Errorf("setup was interrupted before it finished cleaning up; "+
			"remove %s and, for a Docker database, run 'docker compose down -v' there", m.generationRoot)
	default:
		return m.generation.err
	}
}

func (m *Model) cancelAndWait() tea.Cmd {
	m.Cancelling = true
	m.ProgressStatus = "Cancelling: stopping child processes and removing the partial project..."
	m.CancelGeneration()

	// No deadline of its own: the scaffold bounds its rollback (see
	// scaffold.rollbackTimeout), and quitting first would end the process
	// with the partial project and its database container still in place. A
	// second ctrl+c still forces the quit, and Err then says what was left.
	finished := m.generation.done
	return func() tea.Msg {
		<-finished
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
