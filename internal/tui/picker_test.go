package tui

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/vaibdix/django-stack/internal/command"
	"github.com/vaibdix/django-stack/internal/command/commandtest"
	"github.com/vaibdix/django-stack/internal/config"
	"github.com/vaibdix/django-stack/internal/pyenv"
	"github.com/vaibdix/django-stack/internal/pypi"
	"github.com/vaibdix/django-stack/internal/releases"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/x/exp/teatest"
)

func TestAnUnreachablePyPIIsReportedInsteadOfAnsweredFromABundledList(t *testing.T) {
	t.Parallel()

	options, note := djangoReleaseOptions(context.Background(), unreachableCatalog(), pyenv.Option{Major: 3, Minor: 14, Version: "3.14.3"}, reviewDay)
	if len(options) != 0 {
		t.Fatalf("options = %v, want none: a list bundled at build time only defers the install failure", options)
	}
	if !strings.Contains(note, "network unavailable") || !strings.Contains(note, "retry") {
		t.Fatalf("the note does not give the cause and how to retry: %q", note)
	}
}

var fakePython = pyenv.Option{
	Path:      "/opt/homebrew/bin/python3",
	Version:   "3.14.3",
	Label:     "Python 3.14.3 (/opt/homebrew/bin/python3) [default]",
	IsDefault: true,
	Major:     3,
	Minor:     14,
	Patch:     3,
}

var fakeDjango = releases.Option{
	Version:       "6.0.3",
	Label:         "6.0.3 [recommended] - compatible with Python 3.14",
	Recommended:   true,
	Compatibility: "compatible with Python 3.14",
}

func TestUpdateAdvancesFromProjectNameToPythonToDjangoSelection(t *testing.T) {
	restoreLoaders := withPickerLoaders(
		func(context.Context, command.Runner) ([]pyenv.Option, error) { return []pyenv.Option{fakePython}, nil },
		func(pyenv.Option, time.Time) ([]releases.Option, string) {
			return []releases.Option{fakeDjango}, "Showing the 5 latest compatible stable Django releases from PyPI."
		},
	)
	defer restoreLoaders()

	model := NewModel()

	updatedModel, _ := model.Update(model.discoverPythonCmd()())
	current := mustModel(updatedModel)
	if current.SelectedPythonPath != fakePython.Path {
		t.Fatalf("expected discovery to preselect the default Python, got %q", current.SelectedPythonPath)
	}

	current.Step = StepProjectName
	current.ProjectName = "django-stack"
	current.ProjectNameForm.State = huh.StateCompleted

	updatedModel, _ = current.Update(tea.KeyMsg{Type: tea.KeyEnter})
	current = mustModel(updatedModel)
	if current.Step != StepPythonSelection {
		t.Fatalf("expected StepPythonSelection after project name, got %v", current.Step)
	}

	current.PythonForm.State = huh.StateCompleted
	updatedModel, _ = current.Update(tea.KeyMsg{Type: tea.KeyEnter})
	current = mustModel(updatedModel)
	if current.Step != StepDjangoVersion {
		t.Fatalf("expected StepDjangoVersion after Python selection, got %v", current.Step)
	}

	if current.DjangoVersion != "" {
		t.Fatalf("Django version was set before the release lookup returned: %q", current.DjangoVersion)
	}

	updatedModel, _ = current.Update(current.fetchDjangoOptionsCmd()())
	current = mustModel(updatedModel)
	if current.DjangoVersion != "6.0.3" {
		t.Fatalf("expected Django version to be preselected from compatible options, got %q", current.DjangoVersion)
	}
}

func TestNewModelPerformsNoDiscoveryIO(t *testing.T) {
	pythonCalls, djangoCalls := 0, 0
	restoreLoaders := withPickerLoaders(
		func(context.Context, command.Runner) ([]pyenv.Option, error) {
			pythonCalls++
			return []pyenv.Option{fakePython}, nil
		},
		func(pyenv.Option, time.Time) ([]releases.Option, string) {
			djangoCalls++
			return []releases.Option{fakeDjango}, ""
		},
	)
	defer restoreLoaders()

	m := NewModel()

	if pythonCalls != 0 {
		t.Errorf("NewModel ran interpreter discovery %d time(s); it must be deferred to Init", pythonCalls)
	}
	if djangoCalls != 0 {
		t.Errorf("NewModel fetched Django releases %d time(s); it must be deferred to the version step", djangoCalls)
	}
	if m.PythonForm == nil || m.DjangoVersionForm == nil {
		t.Fatal("NewModel must still produce usable pending forms")
	}
}

func TestCompatibilityRulesDeclareUsablePythonRanges(t *testing.T) {
	t.Parallel()

	for _, rule := range config.DjangoCompatibilityRules {
		if rule.MinMinor > rule.MaxMinor {
			t.Errorf("Django %s declares Python 3.%d-3.%d, an empty range",
				rule.Series, rule.MinMinor, rule.MaxMinor)
		}
	}
}

func withPickerLoaders(
	pythonLoader func(context.Context, command.Runner) ([]pyenv.Option, error),
	djangoLoader func(pyenv.Option, time.Time) ([]releases.Option, string),
) func() {
	originalPython := pyenv.Load
	originalDjango := fetchDjangoReleaseOptions
	pyenv.Load = pythonLoader
	fetchDjangoReleaseOptions = func(_ context.Context, _ *releases.Catalog, option pyenv.Option, today time.Time) ([]releases.Option, string) {
		return djangoLoader(option, today)
	}
	return func() {
		pyenv.Load = originalPython
		fetchDjangoReleaseOptions = originalDjango
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func unreachableCatalog() *releases.Catalog {
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("network unavailable")
	})}
	return releases.New(client, func(context.Context) string { return pypi.DefaultIndex })
}

func TestTheWizardStopsWithoutAPythonOrADjangoVersion(t *testing.T) {
	m := questionModel(t, StepPythonSelection)
	m.applyPythonOptions(nil, nil)
	m.PythonForm.Init()
	m = settle(m, func() tea.Cmd { _, c := m.Update(tea.KeyMsg{Type: tea.KeyEnter}); return c }())
	if m.Step != StepPythonSelection {
		t.Errorf("Enter with no interpreter moved on to %v", m.Step)
	}

	m = questionModel(t, StepDjangoVersion)
	m.applyDjangoOptions(nil, "Choose a Python interpreter first.")
	m.DjangoVersionForm.Init()
	m = settle(m, func() tea.Cmd { _, c := m.Update(tea.KeyMsg{Type: tea.KeyEnter}); return c }())
	if m.Step != StepDjangoVersion {
		t.Errorf("Enter with no Django version moved on to %v", m.Step)
	}
}

func TestAnInterpreterNewerThanEveryDjangoReleaseIsExplainedNotListedEmpty(t *testing.T) {
	t.Parallel()

	options, note := djangoReleaseOptions(context.Background(), unreachableCatalog(), pyenv.Option{Major: 3, Minor: 15, Version: "3.15.0"}, reviewDay)
	if len(options) != 0 {
		t.Errorf("Python 3.15 was offered %v, which Django does not support yet", options)
	}
	if !strings.Contains(note, "3.15") || !strings.Contains(note, "older interpreter") {
		t.Errorf("the note does not explain what to do: %q", note)
	}
}

var reviewDay = time.Date(2026, time.October, 16, 0, 0, 0, 0, time.UTC)

func TestALateReleaseListForAnEarlierInterpreterIsIgnored(t *testing.T) {
	older := pyenv.Option{Path: "/a/python3.11", Version: "3.11.9", Major: 3, Minor: 11, Patch: 9}
	newer := pyenv.Option{Path: "/b/python3.13", Version: "3.13.13", Major: 3, Minor: 13, Patch: 13}
	restore := withPickerLoaders(
		func(context.Context, command.Runner) ([]pyenv.Option, error) {
			return []pyenv.Option{older, newer}, nil
		},
		func(option pyenv.Option, _ time.Time) ([]releases.Option, string) {
			if option.Path == older.Path {
				return []releases.Option{{Version: "5.2.18", Label: "5.2.18"}}, "older"
			}
			return []releases.Option{{Version: "6.1.2", Label: "6.1.2"}}, "newer"
		},
	)
	defer restore()

	m := questionModel(t, StepDjangoVersion)
	m.AvailablePythonOptions = []pyenv.Option{older, newer}
	m.SelectedPythonPath = older.Path
	staleFetch := m.fetchDjangoOptionsCmd()

	m.SelectedPythonPath = newer.Path
	currentFetch := m.fetchDjangoOptionsCmd()

	updated, _ := m.Update(currentFetch())
	m = mustModel(updated)
	updated, _ = m.Update(staleFetch())
	m = mustModel(updated)

	if m.DjangoVersion != "6.1.2" || m.DjangoVersionNote != "newer" {
		t.Errorf("the release list for %s replaced the one for %s: version %q, note %q",
			older.Path, newer.Path, m.DjangoVersion, m.DjangoVersionNote)
	}
}

type replyRecorder struct {
	model   *Model
	replies chan djangoOptionsMsg
}

func (r replyRecorder) Init() tea.Cmd { return r.model.Init() }

func (r replyRecorder) View() string { return r.model.View() }

func (r replyRecorder) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	updated, cmd := r.model.Update(msg)
	r.model = mustModel(updated)
	if reply, ok := msg.(djangoOptionsMsg); ok {
		r.replies <- reply
	}
	return r, cmd
}

func TestGoingBackDuringTheReleaseFetchShowsTheNewInterpretersList(t *testing.T) {
	older := pyenv.Option{Path: "/a/python3.11", Label: "Python 3.11.9 older", Version: "3.11.9", Major: 3, Minor: 11, Patch: 9}
	newer := pyenv.Option{Path: "/b/python3.13", Label: "Python 3.13.13 newer", Version: "3.13.13", Major: 3, Minor: 13, Patch: 13}
	releaseOlder := make(chan struct{})
	restore := withPickerLoaders(
		func(context.Context, command.Runner) ([]pyenv.Option, error) {
			return []pyenv.Option{older, newer}, nil
		},
		func(option pyenv.Option, _ time.Time) ([]releases.Option, string) {
			if option.Path == older.Path {
				<-releaseOlder
				return []releases.Option{{Version: "5.2.18", Label: "Django 5.2.18 for the older interpreter"}}, "older"
			}
			return []releases.Option{{Version: "6.1.2", Label: "Django 6.1.2 for the newer interpreter"}}, "newer"
		},
	)
	defer restore()

	m := NewModel()
	m.Runner = commandtest.NewFake()
	recorder := replyRecorder{model: m, replies: make(chan djangoOptionsMsg, 4)}
	tm := teatest.NewTestModel(t, recorder, teatest.WithInitialTermSize(100, 32))
	defer func() {
		select {
		case <-releaseOlder:
		default:
			close(releaseOlder)
		}
	}()

	shows := func(text string) {
		t.Helper()
		teatest.WaitFor(t, tm.Output(), func(out []byte) bool { return bytes.Contains(out, []byte(text)) },
			teatest.WithDuration(10*time.Second))
	}
	reply := func() djangoOptionsMsg {
		t.Helper()
		select {
		case msg := <-recorder.replies:
			return msg
		case <-time.After(10 * time.Second):
			t.Fatal("no Django release reply reached Update")
			return djangoOptionsMsg{}
		}
	}

	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	shows("Project Name")
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	shows(newer.Label)
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	shows("Fetching the latest compatible")

	tm.Send(tea.KeyMsg{Type: tea.KeyShiftTab})
	shows(newer.Label)
	tm.Send(tea.KeyMsg{Type: tea.KeyDown})
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	if got := reply(); got.Interpreter != newer.Path {
		t.Fatalf("the first reply was for %s, want %s", got.Interpreter, newer.Path)
	}
	shows("Django 6.1.2 for the newer interpreter")

	close(releaseOlder)
	if got := reply(); got.Interpreter != older.Path {
		t.Fatalf("the late reply was for %s, want %s", got.Interpreter, older.Path)
	}

	if err := tm.Quit(); err != nil {
		t.Fatalf("Quit: %v", err)
	}
	final, ok := tm.FinalModel(t, teatest.WithFinalTimeout(10*time.Second)).(replyRecorder)
	if !ok {
		t.Fatal("the final model is not the recorder")
	}
	got := final.model
	if got.Step != StepDjangoVersion || got.SelectedPythonPath != newer.Path {
		t.Fatalf("step %v with interpreter %s, want the Django step for %s", got.Step, got.SelectedPythonPath, newer.Path)
	}
	if got.DjangoVersion != "6.1.2" || got.DjangoVersionNote != "newer" {
		t.Errorf("the late list for %s replaced the one for %s: version %q, note %q",
			older.Path, newer.Path, got.DjangoVersion, got.DjangoVersionNote)
	}
	if view := got.View(); strings.Contains(view, "5.2.18") || !strings.Contains(view, "6.1.2") {
		t.Errorf("the Django screen shows the wrong list:\n%s", view)
	}
}
