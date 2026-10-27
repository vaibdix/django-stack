package tui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	tea "charm.land/bubbletea/v2"

	"github.com/vaibdix/django-stack/internal/config"
	"github.com/vaibdix/django-stack/internal/event"
	"github.com/vaibdix/django-stack/internal/render"
	"github.com/vaibdix/django-stack/internal/scaffold"
)

type scaffoldStepMsg string

type scaffoldProgressMsg string

type scaffoldFinishedMsg struct{}

type scaffoldDoneMsg struct{ Err error }

type toolsDecidedMsg render.Tools

type sender struct {
	events chan<- tea.Msg
	done   <-chan struct{}
}

func (r sender) send(msg tea.Msg) {
	select {
	case r.events <- msg:
	case <-r.done:
	}
}

func message(ev event.Event) tea.Msg {
	switch ev := ev.(type) {
	case event.Note:
		return scaffoldStepMsg(ev.Text)
	case event.Started:
		return scaffoldProgressMsg(ev.Label)
	case event.Finished:
		return scaffoldFinishedMsg{}
	default:
		return nil
	}
}

func (r sender) forward(engine <-chan event.Event) {
	for ev := range engine {
		if msg := message(ev); msg != nil {
			r.send(msg)
		}
	}
}

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

func projectRoot(name string) (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return filepath.Join(wd, name), nil
}

func (m *Model) startGeneration() tea.Cmd {
	root, err := projectRoot(m.choices.Name)
	if err != nil {
		return func() tea.Msg { return scaffoldDoneMsg{Err: err} }
	}
	m.spec, _, err = config.Parse(m.choices, m.interpreter(), m.clock())
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

	tui := sender{events: events, done: ctx.Done()}
	engine := make(chan event.Event, scaffoldEventBuffer)
	forwarded := make(chan struct{})
	go func() {
		defer close(forwarded)
		tui.forward(engine)
	}()
	spec, resolved, opts := m.spec, m.resolved, m.engine
	opts.Root, opts.Events = root, engine
	opts.Ports.Run, opts.Ports.Clock = m.commandRunner(), m.clock

	go func() {
		defer close(run.done)
		defer close(events)
		run.err = generate(ctx, spec, resolved, opts, tui)
		close(engine)
		<-forwarded
		tui.send(scaffoldDoneMsg{Err: run.err})
	}()

	return m.waitForScaffoldEvent()
}

func generate(ctx context.Context, spec config.Spec, resolved config.Resolved, opts scaffold.Options, tui sender) error {
	tools, err := scaffold.Preflight(ctx, spec, resolved, opts.Ports)
	if err != nil {
		return err
	}
	tui.send(toolsDecidedMsg(tools))
	opts.Toolchain, opts.NodeManager = tools.Toolchain, tools.NodeManager
	return scaffold.New(spec, resolved, opts).Generate(ctx)
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

type devEnvStartedMsg []string

func (m *Model) startDevelopmentEnvironment() tea.Cmd {
	spec, resolved, opts := m.spec, m.resolved, m.engine
	opts.Ports.Run = m.commandRunner()
	return func() tea.Msg {
		root, err := projectRoot(spec.Choices().Name)
		if err != nil {
			return devEnvStartedMsg{"Could not resolve the project directory: " + err.Error()}
		}
		events := make(chan event.Event, scaffoldEventBuffer)
		collected := make(chan []string)
		go func() { collected <- notes(events) }()
		opts.Root, opts.Events = root, events
		scaffold.New(spec, resolved, opts).StartDevelopmentEnvironment(context.Background())
		close(events)
		return devEnvStartedMsg(<-collected)
	}
}

func notes(events <-chan event.Event) []string {
	var lines []string
	for ev := range events {
		switch ev := ev.(type) {
		case event.Note:
			lines = append(lines, ev.Text)
		case event.Started:
			lines = append(lines, ev.Label)
		}
	}
	return lines
}
