package scaffold

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/vaibdix/django-stack/internal/event"
	"github.com/vaibdix/django-stack/internal/plan"
	"github.com/vaibdix/django-stack/internal/port"
)

func recordMessages(e *Env) func() []string {
	events := make(chan event.Event, 1024)
	e.events = events
	return func() []string {
		close(events)
		var messages []string
		for ev := range events {
			switch ev := ev.(type) {
			case event.Note:
				messages = append(messages, ev.Text)
			case event.Started:
				messages = append(messages, ev.Label)
			case event.Output:
				messages = append(messages, ev.Line)
			}
		}
		return messages
	}
}

type linesRunner struct{ lines []string }

func (r linesRunner) Run(context.Context, string, string, ...string) ([]byte, error) {
	return []byte(strings.Join(r.lines, "\n")), nil
}

func (r linesRunner) RunLines(_ context.Context, _ string, line func(string), _ string, _ ...string) ([]byte, error) {
	for _, l := range r.lines {
		line(l)
	}
	return []byte(strings.Join(r.lines, "\n")), nil
}

func TestCommandOutputIsSentLineByLineUnderItsEffect(t *testing.T) {
	t.Parallel()

	e := &Env{ports: port.Ports{Run: linesRunner{lines: []string{"Resolved 12 packages", "Installed 12 packages"}}}}
	events := make(chan event.Event, 8)
	e.events = events
	ctx := withEffect(t.Context(), plan.Effect{ID: plan.InstallPackages})
	if _, err := e.runIn(ctx, t.TempDir(), "uv", "pip", "install"); err != nil {
		t.Fatalf("runIn: %v", err)
	}
	if _, err := e.runIn(t.Context(), t.TempDir(), "uv", "--version"); err != nil {
		t.Fatalf("runIn: %v", err)
	}
	close(events)
	var got []event.Event
	for ev := range events {
		got = append(got, ev)
	}
	want := []event.Event{
		event.Output{Effect: plan.InstallPackages, Line: "Resolved 12 packages"},
		event.Output{Effect: plan.InstallPackages, Line: "Installed 12 packages"},
	}
	if !slices.Equal(got, want) {
		t.Errorf("events = %#v, want %#v; a command outside an effect sends nothing", got, want)
	}
}
