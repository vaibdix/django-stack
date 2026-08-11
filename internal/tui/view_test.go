package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
)

// newModelForViewTest builds a Model with just enough widget state for View to
// render. NewModel cannot be used here: it shells out to discover Python
// interpreters and queries PyPI for Django releases, so it needs a network and
// a populated machine.
func newModelForViewTest() *Model {
	return &Model{
		Spinner:  spinner.New(),
		Progress: progress.New(progress.WithWidth(40)),
		Width:    100,
	}
}

// RecentStepMessages replaced an inline slice in View that initialised its start
// index to len(StepMessages) and only clamped when len > 3. For 1, 2 or 3
// messages the resulting slice was empty, so the setup screen rendered a
// "Recent steps:" header with nothing under it — the entire progress log was
// invisible until the fourth step.

func TestRecentStepMessagesReturnsUpToNMostRecent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		messages []string
		n        int
		want     []string
	}{
		{name: "empty", messages: nil, n: 3, want: nil},
		{name: "one message", messages: []string{"a"}, n: 3, want: []string{"a"}},
		{name: "two messages", messages: []string{"a", "b"}, n: 3, want: []string{"a", "b"}},
		{name: "exactly n", messages: []string{"a", "b", "c"}, n: 3, want: []string{"a", "b", "c"}},
		{name: "more than n keeps the last n", messages: []string{"a", "b", "c", "d", "e"}, n: 3, want: []string{"c", "d", "e"}},
		{name: "n of one", messages: []string{"a", "b", "c"}, n: 1, want: []string{"c"}},
		{name: "n of zero", messages: []string{"a", "b"}, n: 0, want: nil},
		{name: "negative n", messages: []string{"a", "b"}, n: -1, want: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			m := &Model{}
			m.StepMessages = tt.messages

			got := m.RecentStepMessages(tt.n)

			if len(got) != len(tt.want) {
				t.Fatalf("RecentStepMessages(%d) returned %d messages %q, want %d %q",
					tt.n, len(got), got, len(tt.want), tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("message %d = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

// The returned slice must not alias the model's storage, or a caller could
// mutate step history and a concurrent append could be observed mid-flight.
func TestRecentStepMessagesReturnsACopy(t *testing.T) {
	t.Parallel()

	m := &Model{}
	m.StepMessages = []string{"first", "second"}

	got := m.RecentStepMessages(2)
	got[0] = "mutated"

	if m.StepMessages[0] != "first" {
		t.Errorf("mutating the returned slice changed the model: StepMessages[0] = %q, want %q",
			m.StepMessages[0], "first")
	}
}

// View must not mutate the model. The progress width clamp used to live in the
// StepSetup branch, duplicating logic already in Update's WindowSizeMsg handler.
func TestViewDoesNotMutateProgressWidth(t *testing.T) {
	t.Parallel()

	m := newModelForViewTest()
	m.Step = StepSetup
	m.Width = 200
	m.Progress.Width = 42

	_ = m.View()

	if m.Progress.Width != 42 {
		t.Errorf("View() changed Progress.Width to %d; View must be a pure function of model state",
			m.Progress.Width)
	}
}

// The setup screen must show step messages as soon as the first one arrives.
func TestViewShowsStepMessagesFromTheFirstOne(t *testing.T) {
	t.Parallel()

	for _, count := range []int{1, 2, 3, 4} {
		m := newModelForViewTest()
		m.Step = StepSetup
		m.Width = 120
		for i := 0; i < count; i++ {
			m.StepMessages = append(m.StepMessages, "step message "+string(rune('A'+i)))
		}

		out := m.View()

		if !strings.Contains(out, "Recent steps:") {
			t.Fatalf("with %d messages, View() omitted the Recent steps header", count)
		}

		newest := "step message " + string(rune('A'+count-1))
		if !strings.Contains(out, newest) {
			t.Errorf("with %d messages, View() did not render the newest message %q.\n"+
				"This is the off-by-one that hid the whole progress log for the first three steps.",
				count, newest)
		}
	}
}
