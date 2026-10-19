package tui

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/vaibdix/django-cli/internal/config"
	"github.com/vaibdix/django-cli/internal/pypi"
	"github.com/vaibdix/django-cli/internal/scaffold/command"
	"github.com/vaibdix/django-cli/internal/scaffold/pyenv"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
)

func TestBuildDjangoOptionsFromReleaseIndexFiltersAndSortsByCompatibility(t *testing.T) {
	t.Parallel()

	releases := []pypi.Release{
		{Version: "6.0.3", UploadedAt: mustTime("2026-02-03T10:00:00Z")},
		{Version: "5.2.12", UploadedAt: mustTime("2026-02-03T09:00:00Z")},
		{Version: "5.1.6", UploadedAt: mustTime("2026-01-15T09:00:00Z")},
		{Version: "4.2.28", UploadedAt: mustTime("2025-12-01T09:00:00Z")},
		{Version: "6.0rc1", UploadedAt: mustTime("2026-01-01T09:00:00Z")},
	}

	py314 := pyenv.Option{Major: 3, Minor: 14, Version: "3.14.3"}
	options314 := buildDjangoOptionsFromReleases(py314, releases, reviewDay)
	if len(options314) != 2 {
		t.Fatalf("expected 2 compatible releases for Python 3.14, got %d", len(options314))
	}
	if options314[0].Version != "6.0.3" || options314[1].Version != "5.2.12" {
		t.Fatalf("unexpected release ordering for Python 3.14: %#v", options314)
	}

	py311 := pyenv.Option{Major: 3, Minor: 11, Version: "3.11.14"}
	options311 := buildDjangoOptionsFromReleases(py311, releases, reviewDay)
	expected := []string{"5.2.12"}
	if len(options311) != len(expected) {
		t.Fatalf("expected %d compatible releases for Python 3.11, got %d", len(expected), len(options311))
	}
	for i, version := range expected {
		if options311[i].Version != version {
			t.Fatalf("expected options311[%d] = %s, got %s", i, version, options311[i].Version)
		}
	}
}

func TestAnUnreachablePyPIIsReportedInsteadOfAnsweredFromABundledList(t *testing.T) {
	restoreClient := withDjangoReleaseClient(&http.Client{
		Timeout: time.Second,
		Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return nil, errors.New("network unavailable")
		}),
	})
	defer restoreClient()

	options, note := fetchDjangoReleaseOptionsFromPyPI(pyenv.Option{Major: 3, Minor: 14, Version: "3.14.3"}, reviewDay)
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

var fakeDjango = DjangoOption{
	Version:       "6.0.3",
	Label:         "6.0.3 [recommended] - compatible with Python 3.14",
	Recommended:   true,
	Compatibility: "compatible with Python 3.14",
}

func TestUpdateAdvancesFromProjectNameToPythonToDjangoSelection(t *testing.T) {
	restoreLoaders := withPickerLoaders(
		func(context.Context, command.Runner) ([]pyenv.Option, error) { return []pyenv.Option{fakePython}, nil },
		func(pyenv.Option, time.Time) ([]DjangoOption, string) {
			return []DjangoOption{fakeDjango}, "Showing the 5 latest compatible stable Django releases from PyPI."
		},
	)
	defer restoreLoaders()

	model := NewModel()

	updatedModel, _ := model.Update(model.discoverPythonCmd()())
	current := updatedModel.(*Model)
	if current.SelectedPythonPath != fakePython.Path {
		t.Fatalf("expected discovery to preselect the default Python, got %q", current.SelectedPythonPath)
	}

	current.Step = StepProjectName
	current.ProjectName = "django-stack"
	current.ProjectNameForm.State = huh.StateCompleted

	updatedModel, _ = current.Update(tea.KeyMsg{Type: tea.KeyEnter})
	current = updatedModel.(*Model)
	if current.Step != StepPythonSelection {
		t.Fatalf("expected StepPythonSelection after project name, got %v", current.Step)
	}

	current.PythonForm.State = huh.StateCompleted
	updatedModel, _ = current.Update(tea.KeyMsg{Type: tea.KeyEnter})
	current = updatedModel.(*Model)
	if current.Step != StepDjangoVersion {
		t.Fatalf("expected StepDjangoVersion after Python selection, got %v", current.Step)
	}

	if current.DjangoVersion != "" {
		t.Fatalf("Django version was set before the release lookup returned: %q", current.DjangoVersion)
	}

	updatedModel, _ = current.Update(current.fetchDjangoOptionsCmd()())
	current = updatedModel.(*Model)
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
		func(pyenv.Option, time.Time) ([]DjangoOption, string) {
			djangoCalls++
			return []DjangoOption{fakeDjango}, ""
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

func withDjangoReleaseClient(client *http.Client) func() {
	original := djangoReleaseHTTPClient
	djangoReleaseHTTPClient = client
	return func() { djangoReleaseHTTPClient = original }
}

func withPickerLoaders(
	pythonLoader func(context.Context, command.Runner) ([]pyenv.Option, error),
	djangoLoader func(pyenv.Option, time.Time) ([]DjangoOption, string),
) func() {
	originalPython := pyenv.Load
	originalDjango := fetchDjangoReleaseOptions
	pyenv.Load = pythonLoader
	fetchDjangoReleaseOptions = djangoLoader
	return func() {
		pyenv.Load = originalPython
		fetchDjangoReleaseOptions = originalDjango
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
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

	options, note := fetchDjangoReleaseOptionsFromPyPI(pyenv.Option{Major: 3, Minor: 15, Version: "3.15.0"}, reviewDay)
	if len(options) != 0 {
		t.Errorf("Python 3.15 was offered %v, which Django does not support yet", options)
	}
	if !strings.Contains(note, "3.15") || !strings.Contains(note, "older interpreter") {
		t.Errorf("the note does not explain what to do: %q", note)
	}
}

func TestEachSeriesIsListedOnceWithItsNewestPatch(t *testing.T) {
	t.Parallel()

	releases := []pypi.Release{
		{Version: "6.1.2", UploadedAt: mustTime("2026-10-01T09:00:00Z")},
		{Version: "6.1.1", UploadedAt: mustTime("2026-09-01T09:00:00Z")},
		{Version: "6.0.9", UploadedAt: mustTime("2026-10-01T09:05:00Z")},
		{Version: "5.2.18", UploadedAt: mustTime("2026-10-01T09:10:00Z")},
		{Version: "5.2.17", UploadedAt: mustTime("2026-09-01T09:00:00Z")},
		{Version: "6.2a1", UploadedAt: mustTime("2026-10-02T09:00:00Z")},
	}

	options := buildDjangoOptionsFromReleases(pyenv.Option{Major: 3, Minor: 13, Version: "3.13.13"}, releases, reviewDay)
	got := make([]string, 0, len(options))
	for _, option := range options {
		got = append(got, option.Version)
	}
	if strings.Join(got, ",") != "6.1.2,6.0.9,5.2.18" {
		t.Fatalf("options = %v, want the newest patch of each series, newest series first", got)
	}
	if !options[0].Recommended || options[2].Recommended {
		t.Errorf("the LTS patch uploaded last was recommended over the newest series: %+v", options)
	}
	if !strings.Contains(options[2].Label, "LTS") {
		t.Errorf("5.2 is not labelled LTS: %q", options[2].Label)
	}
}

func mustTime(text string) time.Time {
	parsed, err := time.Parse(time.RFC3339, text)
	if err != nil {
		panic(err)
	}
	return parsed
}

func TestReleasesThatRequireANewerPythonAreSkipped(t *testing.T) {
	t.Parallel()

	releases := []pypi.Release{
		{Version: "6.1.2", RequiresPython: ">=3.12"},
		{Version: "5.2.18", RequiresPython: ">=3.10"},
	}
	options := buildDjangoOptionsFromReleases(pyenv.Option{Major: 3, Minor: 11}, releases, reviewDay)
	if len(options) != 1 || options[0].Version != "5.2.18" {
		t.Errorf("options for Python 3.11 = %+v, want only 5.2.18", options)
	}
}

// reviewDay pins the support table to a date when 5.2, 6.0 and 6.1 are all
// still supported, so these tests do not start failing when a series ends.
var reviewDay = time.Date(2026, time.October, 16, 0, 0, 0, 0, time.UTC)

func TestASeriesPastItsEndOfSupportIsNoLongerOffered(t *testing.T) {
	t.Parallel()

	releases := []pypi.Release{
		{Version: "6.1.2"},
		{Version: "6.0.9"},
		{Version: "5.2.18"},
	}
	afterDjango60 := time.Date(2027, time.June, 1, 0, 0, 0, 0, time.UTC)

	options := buildDjangoOptionsFromReleases(pyenv.Option{Major: 3, Minor: 13, Version: "3.13.13"}, releases, afterDjango60)
	got := make([]string, 0, len(options))
	for _, option := range options {
		got = append(got, option.Version)
	}
	if strings.Join(got, ",") != "6.1.2,5.2.18" {
		t.Fatalf("options on %s = %v, want 6.0 dropped after its April 2027 end of support",
			afterDjango60.Format(time.DateOnly), got)
	}
}
