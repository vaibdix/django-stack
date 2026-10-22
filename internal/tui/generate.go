package tui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/vaibdix/django-stack/internal/config"
	"github.com/vaibdix/django-stack/internal/scaffold"
)

type scaffoldStepMsg string

type scaffoldProgressMsg string

type scaffoldDoneMsg struct{ Err error }

type toolsDecidedMsg scaffold.Tools

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
	return filepath.Join(wd, m.project.ProjectName), nil
}

func (m *Model) startGeneration() tea.Cmd {
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
	cfg, opts := m.project, m.engine
	opts.Root, opts.Runner, opts.Reporter, opts.Clock = root, m.commandRunner(), reporter, m.clock

	go func() {
		defer close(run.done)
		defer close(events)
		run.err = generate(ctx, cfg, opts, reporter)
		reporter.send(scaffoldDoneMsg{Err: run.err})
	}()

	return m.waitForScaffoldEvent()
}

func generate(ctx context.Context, cfg config.Project, opts scaffold.Options, reporter eventReporter) error {
	tools, err := scaffold.Preflight(ctx, cfg, opts.Runner, opts.Clock())
	if err != nil {
		return err
	}
	reporter.send(toolsDecidedMsg(tools))
	opts.Toolchain, opts.NodeManager = tools.Toolchain, tools.NodeManager
	return scaffold.New(cfg, opts).Generate(ctx)
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

func (m *Model) cancelGeneration() {
	if m.cancelScaffold != nil {
		m.cancelScaffold()
	}
}

func (m *Model) generationRunning() bool {
	return m.generation != nil && !m.generation.finished()
}

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
	m.cancelling = true
	m.progressStatus = "Cancelling: stopping child processes and removing the partial project..."
	m.cancelGeneration()

	finished := m.generation.done
	return func() tea.Msg {
		<-finished
		return tea.QuitMsg{}
	}
}

func (m *Model) startDevelopmentEnvironment() {
	root, err := m.projectRoot()
	if err != nil {
		m.stepMessages = append(m.stepMessages, "Could not resolve the project directory: "+err.Error())
		return
	}
	opts := m.engine
	opts.Root, opts.Runner, opts.Reporter = root, m.commandRunner(), directReporter{m}
	scaffold.New(m.project, opts).StartDevelopmentEnvironment()
}

type directReporter struct{ m *Model }

func (r directReporter) Step(message string) {
	r.m.stepMessages = append(r.m.stepMessages, message)
}

func (r directReporter) Progress(status string) {
	r.m.progressStatus = status
}
