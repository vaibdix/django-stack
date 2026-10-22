package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"

	"github.com/vaibdix/django-stack/internal/scaffold"
)

func newModelForViewTest() *Model {
	return &Model{
		Spinner:  spinner.New(),
		Progress: progress.New(progress.WithWidth(40)),
		Width:    100,
		Height:   32,
	}
}

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

func TestViewShowsStepMessagesFromTheFirstOne(t *testing.T) {
	t.Parallel()

	for _, count := range []int{1, 2, 3, 4} {
		m := newModelForViewTest()
		m.Step = StepSetup
		m.Width = 120
		for i := range count {
			m.StepMessages = append(m.StepMessages, "step message "+string(rune('A'+i)))
		}

		out := m.View()

		if !strings.Contains(out, recentStepsLabel) {
			t.Fatalf("with %d messages, View() omitted the %q header", count, recentStepsLabel)
		}

		newest := "step message " + string(rune('A'+count-1))
		if !strings.Contains(out, newest) {
			t.Errorf("with %d messages, View() did not render the newest message %q.\n"+
				"This is the off-by-one that hid the whole progress log for the first three steps.",
				count, newest)
		}
	}
}

func TestAFailedVerificationSaysTheProjectExists(t *testing.T) {
	m := NewModel()
	m.Width, m.Height = 100, 32
	m.ProjectName = "shop"
	m.Error = &scaffold.VerificationError{Check: "test suite", Output: "FAILED (errors=1)"}

	view := m.View()
	if !strings.Contains(view, "The project is in shop") {
		t.Errorf("a verification failure reads as if nothing was generated:\n%s", view)
	}
}
