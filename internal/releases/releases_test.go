package releases

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/vaibdix/django-stack/internal/command/commandtest"
	"github.com/vaibdix/django-stack/internal/pyenv"
	"github.com/vaibdix/django-stack/internal/pypi"
)

var reviewDay = time.Date(2026, time.October, 16, 0, 0, 0, 0, time.UTC)

const djangoIndex = `{
  "meta": {"api-version": "1.4"},
  "name": "django",
  "versions": ["5.2.18", "6.1.2"],
  "files": [
    {"filename": "django-5.2.18-py3-none-any.whl", "requires-python": ">=3.10", "upload-time": "2026-10-06T12:50:00Z"},
    {"filename": "django-6.1.2-py3-none-any.whl", "requires-python": ">=3.12", "upload-time": "2026-10-06T12:53:17Z"}
  ]
}`

func fakeIndex(t *testing.T) (*Catalog, *atomic.Int32) {
	t.Helper()
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if r.URL.Path != "/simple/django/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/vnd.pypi.simple.v1+json")
		_, _ = w.Write([]byte(djangoIndex))
	}))
	t.Cleanup(server.Close)
	return New(server.Client(), func(context.Context) string { return server.URL + "/simple/" }), &requests
}

var (
	python311 = pyenv.Option{Major: 3, Minor: 11, Patch: 9, Version: "3.11.9"}
	python313 = pyenv.Option{Major: 3, Minor: 13, Patch: 13, Version: "3.13.13"}
	python315 = pyenv.Option{Major: 3, Minor: 15, Version: "3.15.0"}
)

func TestTheReleasesAreFetchedOncePerIndex(t *testing.T) {
	t.Parallel()

	catalog, requests := fakeIndex(t)
	for _, python := range []pyenv.Option{python313, python311, python313} {
		if _, err := catalog.Options(context.Background(), python, reviewDay); err != nil {
			t.Fatalf("Options(%s): %v", python.Version, err)
		}
	}
	if got := requests.Load(); got != 1 {
		t.Errorf("going back and forth between interpreters fetched the index %d times, want 1", got)
	}
}

func TestAFailedFetchIsNotCached(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		calls.Add(1)
		return nil, errors.New("network unavailable")
	})}
	catalog := New(client, func(context.Context) string { return pypi.DefaultIndex })
	for range 2 {
		_, err := catalog.Options(context.Background(), python313, reviewDay)
		var unreadable *FetchError
		if !errors.As(err, &unreadable) || unreadable.Index != pypi.DefaultIndex {
			t.Fatalf("err = %v, want a FetchError naming the index", err)
		}
	}
	if calls.Load() < 2 {
		t.Error("a failed fetch was cached, so choosing the interpreter again cannot retry")
	}
}

func TestAnInterpreterNewerThanEverySeriesIsRejectedWithoutAFetch(t *testing.T) {
	t.Parallel()

	catalog, requests := fakeIndex(t)
	_, err := catalog.Options(context.Background(), python315, reviewDay)
	var unsupported *UnsupportedPythonError
	if !errors.As(err, &unsupported) || unsupported.Newest != 14 {
		t.Fatalf("err = %v, want an UnsupportedPythonError naming 3.14", err)
	}
	if requests.Load() != 0 {
		t.Error("the index was fetched for an interpreter no series supports")
	}
}

func TestResolveAcceptsOnlyAReleaseTheIndexServesForThatPython(t *testing.T) {
	t.Parallel()

	catalog, _ := fakeIndex(t)
	cases := []struct {
		python  pyenv.Option
		version string
		want    string
	}{
		{python313, "6.1.2", ""},
		{python311, "5.2.18", ""},
		{python313, "6.1.99", "is not a release on"},
		{python311, "6.1.2", "no supported Django series covers 6.1.2 on Python 3.11"},
		{python315, "6.1.2", "Python 3.15"},
		{python313, "4.2.20", "no supported Django series covers 4.2.20"},
	}
	for _, c := range cases {
		err := catalog.Resolve(context.Background(), c.python, c.version, reviewDay)
		switch {
		case c.want == "" && err != nil:
			t.Errorf("Resolve(%s, %s) = %v, want nil", c.python.Version, c.version, err)
		case c.want != "" && (err == nil || !strings.Contains(err.Error(), c.want)):
			t.Errorf("Resolve(%s, %s) = %v, want an error containing %q", c.python.Version, c.version, err, c.want)
		}
	}
}

func TestResolveRejectsAReleaseWhoseRequiresPythonExcludesTheInterpreter(t *testing.T) {
	t.Parallel()

	catalog := New(nil, func(context.Context) string { return pypi.DefaultIndex })
	catalog.cache[pypi.DefaultIndex] = []pypi.Release{{Version: "5.2.18", RequiresPython: ">=3.12"}}
	err := catalog.Resolve(context.Background(), python311, "5.2.18", reviewDay)
	if err == nil || !strings.Contains(err.Error(), "requires Python >=3.12") {
		t.Errorf("err = %v, want the requires-python mismatch", err)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func TestTheToolchainIsDetectedWithOneProbe(t *testing.T) {
	t.Parallel()

	fake := commandtest.NewFake()
	fake.Responses["uv --version"] = []byte("uv 0.13.1\n")
	if got := DetectToolchain(context.Background(), fake); got != pyenv.UV {
		t.Errorf("uv answering --version gave %v, want uv", got)
	}
	if calls := fake.CommandLines(); len(calls) != 1 || calls[0] != "uv --version" {
		t.Errorf("detection ran %v, want exactly one `uv --version`", calls)
	}

	missing := commandtest.NewFake()
	missing.FailWith("uv --version", "executable file not found in $PATH")
	if got := DetectToolchain(context.Background(), missing); got != pyenv.Pip {
		t.Errorf("a missing uv gave %v, want pip", got)
	}

	silent := commandtest.NewFake()
	if got := DetectToolchain(context.Background(), silent); got != pyenv.Pip {
		t.Errorf("a uv that prints nothing gave %v, want pip", got)
	}
}

func TestTheIndexIsTheOneTheInstallerUses(t *testing.T) {
	t.Parallel()

	env := map[string]string{
		"UV_DEFAULT_INDEX": "https://mirror.example/uv/simple",
		"UV_INDEX_URL":     "https://old.example/simple/",
		"PIP_INDEX_URL":    "https://mirror.example/pip/simple/",
	}
	getenv := func(name string) string { return env[name] }
	if got := IndexFor(pyenv.UV, getenv); got != "https://mirror.example/uv/simple/" {
		t.Errorf("uv index = %q, want UV_DEFAULT_INDEX with a trailing slash", got)
	}
	if got := IndexFor(pyenv.Pip, getenv); got != "https://mirror.example/pip/simple/" {
		t.Errorf("pip index = %q, want PIP_INDEX_URL", got)
	}
	delete(env, "UV_DEFAULT_INDEX")
	if got := IndexFor(pyenv.UV, getenv); got != "https://old.example/simple/" {
		t.Errorf("uv index without UV_DEFAULT_INDEX = %q, want UV_INDEX_URL", got)
	}
	if got := IndexFor(pyenv.UV, func(string) string { return "" }); got != pypi.DefaultIndex {
		t.Errorf("index with nothing configured = %q, want PyPI", got)
	}
}

func TestBuildDjangoOptionsFromReleaseIndexFiltersAndSortsByCompatibility(t *testing.T) {
	t.Parallel()

	index := []pypi.Release{
		{Version: "6.0.3", UploadedAt: mustTime("2026-02-03T10:00:00Z")},
		{Version: "5.2.12", UploadedAt: mustTime("2026-02-03T09:00:00Z")},
		{Version: "5.1.6", UploadedAt: mustTime("2026-01-15T09:00:00Z")},
		{Version: "4.2.28", UploadedAt: mustTime("2025-12-01T09:00:00Z")},
		{Version: "6.0rc1", UploadedAt: mustTime("2026-01-01T09:00:00Z")},
	}

	py314 := pyenv.Option{Major: 3, Minor: 14, Version: "3.14.3"}
	options314 := mustOptions(t)(optionsFrom(py314, index, reviewDay))
	if len(options314) != 2 {
		t.Fatalf("expected 2 compatible releases for Python 3.14, got %d", len(options314))
	}
	if options314[0].Version != "6.0.3" || options314[1].Version != "5.2.12" {
		t.Fatalf("unexpected release ordering for Python 3.14: %#v", options314)
	}

	py311 := pyenv.Option{Major: 3, Minor: 11, Version: "3.11.14"}
	options311 := mustOptions(t)(optionsFrom(py311, index, reviewDay))
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

func TestEachSeriesIsListedOnceWithItsNewestPatch(t *testing.T) {
	t.Parallel()

	index := []pypi.Release{
		{Version: "6.1.2", UploadedAt: mustTime("2026-10-01T09:00:00Z")},
		{Version: "6.1.1", UploadedAt: mustTime("2026-09-01T09:00:00Z")},
		{Version: "6.0.9", UploadedAt: mustTime("2026-10-01T09:05:00Z")},
		{Version: "5.2.18", UploadedAt: mustTime("2026-10-01T09:10:00Z")},
		{Version: "5.2.17", UploadedAt: mustTime("2026-09-01T09:00:00Z")},
		{Version: "6.2a1", UploadedAt: mustTime("2026-10-02T09:00:00Z")},
	}

	options := mustOptions(t)(optionsFrom(pyenv.Option{Major: 3, Minor: 13, Version: "3.13.13"}, index, reviewDay))
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

func TestReleasesThatRequireANewerPythonAreSkipped(t *testing.T) {
	t.Parallel()

	index := []pypi.Release{
		{Version: "6.1.2", RequiresPython: ">=3.12"},
		{Version: "5.2.18", RequiresPython: ">=3.10"},
	}
	options := mustOptions(t)(optionsFrom(pyenv.Option{Major: 3, Minor: 11}, index, reviewDay))
	if len(options) != 1 || options[0].Version != "5.2.18" {
		t.Errorf("options for Python 3.11 = %+v, want only 5.2.18", options)
	}
}

func TestASeriesPastItsEndOfSupportIsNoLongerOffered(t *testing.T) {
	t.Parallel()

	index := []pypi.Release{
		{Version: "6.1.2"},
		{Version: "6.0.9"},
		{Version: "5.2.18"},
	}
	afterDjango60 := time.Date(2027, time.June, 1, 0, 0, 0, 0, time.UTC)

	options := mustOptions(t)(optionsFrom(pyenv.Option{Major: 3, Minor: 13, Version: "3.13.13"}, index, afterDjango60))
	got := make([]string, 0, len(options))
	for _, option := range options {
		got = append(got, option.Version)
	}
	if strings.Join(got, ",") != "6.1.2,5.2.18" {
		t.Fatalf("options on %s = %v, want 6.0 dropped after its April 2027 end of support",
			afterDjango60.Format(time.DateOnly), got)
	}
}

func mustOptions(t *testing.T) func([]Option, error) []Option {
	return func(options []Option, err error) []Option {
		t.Helper()
		if err != nil {
			t.Fatalf("optionsFrom: %v", err)
		}
		return options
	}
}

func mustTime(text string) time.Time {
	parsed, err := time.Parse(time.RFC3339, text)
	if err != nil {
		panic(err)
	}
	return parsed
}
