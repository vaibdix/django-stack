package tui

import (
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/vaibdix/django-cli/internal/scaffold/pyenv"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
)

func TestBuildDjangoOptionsFromReleaseIndexFiltersAndSortsByCompatibility(t *testing.T) {
	t.Parallel()

	releases := map[string][]djangoReleaseFile{
		"6.0.3":  {{UploadTimeISO8601: "2026-02-03T10:00:00Z"}},
		"5.2.12": {{UploadTimeISO8601: "2026-02-03T09:00:00Z"}},
		"5.1.6":  {{UploadTimeISO8601: "2026-01-15T09:00:00Z"}},
		"4.2.28": {{UploadTimeISO8601: "2025-12-01T09:00:00Z"}},
		"6.0rc1": {{UploadTimeISO8601: "2026-01-01T09:00:00Z"}},
	}

	py314 := pyenv.Option{Major: 3, Minor: 14, Version: "3.14.3"}
	options314 := buildDjangoOptionsFromReleaseIndex(py314, releases)
	if len(options314) != 2 {
		t.Fatalf("expected 2 compatible releases for Python 3.14, got %d", len(options314))
	}
	if options314[0].Version != "6.0.3" || options314[1].Version != "5.2.12" {
		t.Fatalf("unexpected release ordering for Python 3.14: %#v", options314)
	}

	py311 := pyenv.Option{Major: 3, Minor: 11, Version: "3.11.14"}
	options311 := buildDjangoOptionsFromReleaseIndex(py311, releases)
	expected := []string{"5.2.12", "5.1.6", "4.2.28"}
	if len(options311) != len(expected) {
		t.Fatalf("expected %d compatible releases for Python 3.11, got %d", len(expected), len(options311))
	}
	for i, version := range expected {
		if options311[i].Version != version {
			t.Fatalf("expected options311[%d] = %s, got %s", i, version, options311[i].Version)
		}
	}
}

func TestFallbackDjangoOptionsForPython314ReturnsLatestCompatibleStableReleases(t *testing.T) {
	t.Parallel()

	py314 := pyenv.Option{Major: 3, Minor: 14, Version: "3.14.3"}
	options := fallbackDjangoOptionsForPython(py314)
	expected := []string{"6.0.3", "5.2.12", "6.0.2", "5.2.11", "6.0.1"}

	if len(options) != len(expected) {
		t.Fatalf("expected %d fallback options, got %d", len(expected), len(options))
	}
	for i, version := range expected {
		if options[i].Version != version {
			t.Fatalf("expected fallback[%d] = %s, got %s", i, version, options[i].Version)
		}
	}
}

func TestFetchDjangoReleaseOptionsFallsBackWhenPyPIIsUnavailable(t *testing.T) {
	restoreClient := withDjangoReleaseClient(&http.Client{
		Timeout: time.Second,
		Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return nil, errors.New("network unavailable")
		}),
	})
	defer restoreClient()

	options, note := fetchDjangoReleaseOptionsFromPyPI(pyenv.Option{Major: 3, Minor: 14, Version: "3.14.3"})
	if len(options) == 0 {
		t.Fatalf("expected bundled fallback options when PyPI is unavailable")
	}
	if !strings.HasPrefix(note, "Showing bundled") {
		t.Fatalf("expected bundled fallback note, got %q", note)
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
		func() []pyenv.Option { return []pyenv.Option{fakePython} },
		func(pyenv.Option) ([]DjangoOption, string) {
			return []DjangoOption{fakeDjango}, "Showing the 5 latest compatible stable Django releases from PyPI."
		},
	)
	defer restoreLoaders()

	model := NewModel()

	updatedModel, _ := model.Update(discoverPythonCmd()())
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
		func() []pyenv.Option { pythonCalls++; return []pyenv.Option{fakePython} },
		func(pyenv.Option) ([]DjangoOption, string) {
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

func TestEveryBundledReleaseHasACompatibilityRule(t *testing.T) {
	t.Parallel()

	known := make(map[string]bool, len(djangoCompatibilityRules))
	for _, rule := range djangoCompatibilityRules {
		known[rule.Series] = true
	}

	for _, version := range fallbackDjangoReleases {
		series := djangoSeries(version)
		if !known[series] {
			t.Errorf("fallbackDjangoReleases contains %s but djangoCompatibilityRules has no rule for the %s series; "+
				"the picker would never offer it", version, series)
		}
	}
}

func TestEveryCompatibilityRuleHasABundledRelease(t *testing.T) {
	t.Parallel()

	bundled := make(map[string]bool, len(fallbackDjangoReleases))
	for _, version := range fallbackDjangoReleases {
		bundled[djangoSeries(version)] = true
	}

	for _, rule := range djangoCompatibilityRules {
		if !bundled[rule.Series] {
			t.Errorf("djangoCompatibilityRules declares the %s series but fallbackDjangoReleases ships no %s release; "+
				"offline users can never select it", rule.Series, rule.Series)
		}
	}
}

func TestBundledReleasesAreStableVersions(t *testing.T) {
	t.Parallel()

	if len(fallbackDjangoReleases) == 0 {
		t.Fatal("fallbackDjangoReleases is empty; offline users would have nothing to select")
	}

	for _, version := range fallbackDjangoReleases {
		if !isStableDjangoVersion(version) {
			t.Errorf("fallbackDjangoReleases contains %q, which is not a stable release", version)
		}
	}
}

func TestCompatibilityRulesDeclareUsablePythonRanges(t *testing.T) {
	t.Parallel()

	for _, rule := range djangoCompatibilityRules {
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
	pythonLoader func() []pyenv.Option,
	djangoLoader func(pyenv.Option) ([]DjangoOption, string),
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
