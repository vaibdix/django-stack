package pypi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

const djangoSimpleFixture = `{
  "meta": {"api-version": "1.4"},
  "name": "django",
  "versions": ["5.0.5", "5.2.18", "6.1rc1", "6.1.2"],
  "files": [
    {"filename": "Django-5.0.5-py3-none-any.whl", "requires-python": ">=3.10", "upload-time": "2024-05-07T07:31:43.123456Z", "yanked": "security"},
    {"filename": "Django-5.0.5.tar.gz", "requires-python": ">=3.10", "upload-time": "2024-05-07T07:31:45.123456Z", "yanked": true},
    {"filename": "django-5.2.18-py3-none-any.whl", "requires-python": ">=3.10", "upload-time": "2026-10-06T12:50:00.000001Z", "yanked": false},
    {"filename": "django-5.2.18.tar.gz", "requires-python": ">=3.10", "upload-time": "2026-10-06T12:51:00.000001Z", "yanked": false},
    {"filename": "django-6.1rc1-py3-none-any.whl", "requires-python": ">=3.12", "upload-time": "2026-07-20T12:00:00Z", "yanked": false},
    {"filename": "django-6.1.2-py3-none-any.whl", "requires-python": ">=3.12", "upload-time": "2026-10-06T12:53:17.381365Z", "yanked": false}
  ]
}`

func TestFetchReadsTheJSONSimpleAPI(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/simple/django/" {
			http.NotFound(w, r)
			return
		}
		if !strings.Contains(r.Header.Get("Accept"), "application/vnd.pypi.simple.v1+json") {
			http.Error(w, "the JSON form of the simple API was not requested", http.StatusNotAcceptable)
			return
		}
		w.Header().Set("Content-Type", "application/vnd.pypi.simple.v1+json")
		_, _ = w.Write([]byte(djangoSimpleFixture))
	}))
	t.Cleanup(server.Close)

	client := server.Client()
	client.Transport = rewriteHost{base: server.URL, next: client.Transport}

	releases, err := Fetch(context.Background(), client, DefaultIndex, "django")
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}

	got := map[string]Release{}
	for _, release := range releases {
		got[release.Version] = release
	}
	if _, yanked := got["5.0.5"]; yanked {
		t.Error("a release whose every file is yanked was returned")
	}
	if len(got) != 3 {
		t.Errorf("releases = %v, want 5.2.18, 6.1rc1 and 6.1.2", releases)
	}
	if got["6.1.2"].RequiresPython != ">=3.12" {
		t.Errorf("requires-python was not carried: %+v", got["6.1.2"])
	}
	if got["5.2.18"].UploadedAt.Minute() != 51 {
		t.Errorf("the release time is not the latest file's upload time: %v", got["5.2.18"].UploadedAt)
	}
}

type rewriteHost struct {
	base string
	next http.RoundTripper
}

func (r rewriteHost) RoundTrip(req *http.Request) (*http.Response, error) {
	clone := req.Clone(req.Context())
	target, err := http.NewRequestWithContext(req.Context(), req.Method, r.base+req.URL.Path, nil)
	if err != nil {
		return nil, err
	}
	clone.URL = target.URL
	clone.Host = target.URL.Host
	return r.next.RoundTrip(clone)
}

func TestSupportsPython(t *testing.T) {
	t.Parallel()

	cases := []struct {
		spec   string
		python []int
		want   bool
	}{
		{">=3.12", []int{3, 13, 0}, true},
		{">=3.12", []int{3, 11, 9}, false},
		{">=3.10, <4", []int{3, 10, 0}, true},
		{"", []int{3, 8, 0}, true},
		{">=3.10, <3.13", []int{3, 13, 1}, false},
		{">=3.10, <3.13", []int{3, 12, 7}, true},
		{"!=3.11.*", []int{3, 11, 4}, false},
		{"!=3.11.*", []int{3, 12, 0}, true},
		{"==3.*", []int{3, 14, 0}, true},
		{"~=3.10", []int{3, 14, 0}, true},
		{"~=3.10.2", []int{3, 11, 0}, false},
		{">3.12", []int{3, 12, 0}, false},
		{"<=3.12", []int{3, 12, 0}, true},
		{">=3.8.1", []int{3, 8, 0}, false},
	}
	for _, tc := range cases {
		got, err := SupportsPython(tc.spec, tc.python...)
		if err != nil || got != tc.want {
			t.Errorf("SupportsPython(%q, %v) = %v, %v; want %v", tc.spec, tc.python, got, err, tc.want)
		}
	}

	for _, bad := range []string{"3.12", ">=three", ">=3.*", "~=3"} {
		if _, err := SupportsPython(bad, 3, 13, 0); err == nil {
			t.Errorf("SupportsPython(%q) read a clause it cannot evaluate", bad)
		}
	}
}

func TestFileHashesCoversEveryFileOfTheVersionAndOnlyThose(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.pypi.simple.v1+json")
		_, _ = w.Write([]byte(`{"versions":["0.11.2","0.14.0"],"files":[
			{"filename":"django-environ-0.11.2.tar.gz","hashes":{"sha256":"old"}},
			{"filename":"django_environ-0.14.0-py3-none-any.whl","hashes":{"sha256":"wheel"}},
			{"filename":"django_environ-0.14.0.tar.gz","hashes":{"sha256":"sdist"}}
		]}`))
	}))
	defer server.Close()

	client := server.Client()
	client.Transport = rewriteHost{base: server.URL, next: client.Transport}

	hashes, err := FileHashes(context.Background(), client, DefaultIndex, "django-environ", "0.14.0")
	if err != nil {
		t.Fatalf("FileHashes: %v", err)
	}
	if strings.Join(hashes, ",") != "sha256:sdist,sha256:wheel" {
		t.Errorf("hashes = %v, want the wheel and the sdist of 0.14.0 only", hashes)
	}

	if _, err := FileHashes(context.Background(), client, DefaultIndex, "django-environ", "9.9.9"); err == nil {
		t.Error("a version PyPI does not list must be an error, not an empty hash list pip would reject")
	}
}

func TestAMirrorIsReadAtItsOwnURL(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/mirror/simple/django/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/vnd.pypi.simple.v1+json")
		_, _ = w.Write([]byte(djangoSimpleFixture))
	}))
	t.Cleanup(server.Close)

	releases, err := Fetch(context.Background(), server.Client(), server.URL+"/mirror/simple/", "django")
	if err != nil || len(releases) == 0 {
		t.Fatalf("Fetch from the mirror = %v, %v; want its releases", releases, err)
	}
}

func TestAnIndexWithoutTheJSONAPIIsReportedNotParsed(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("<html><a href='django-6.1.2.tar.gz'>django-6.1.2.tar.gz</a></html>"))
	}))
	t.Cleanup(server.Close)

	_, err := FileHashes(context.Background(), server.Client(), server.URL+"/simple/", "django", "6.1.2")
	if err == nil || !strings.Contains(err.Error(), "PEP 691") {
		t.Errorf("FileHashes from an HTML-only index = %v, want an error naming PEP 691", err)
	}
}

func TestATransientIndexFailureIsRetried(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if calls.Add(1) <= 2 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/vnd.pypi.simple.v1+json")
		_, _ = w.Write([]byte(`{"versions":["1.0"],"files":[{"filename":"demo-1.0.tar.gz","hashes":{"sha256":"abc"}}]}`))
	}))
	defer server.Close()

	hashes, err := FileHashes(t.Context(), server.Client(), server.URL+"/", "demo", "1.0")
	if err != nil {
		t.Fatalf("FileHashes after two 503s: %v", err)
	}
	if len(hashes) != 1 || calls.Load() != 3 {
		t.Errorf("hashes = %v after %d calls, want one hash after 3", hashes, calls.Load())
	}
}

func TestAPermanentIndexFailureIsNotRetried(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	if _, err := FileHashes(t.Context(), server.Client(), server.URL+"/", "demo", "1.0"); err == nil {
		t.Fatal("a 404 was accepted")
	}
	if calls.Load() != 1 {
		t.Errorf("a 404 was requested %d times, want once", calls.Load())
	}
}

func TestRetriesStopWhenTheContextIsCancelled(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
	defer cancel()
	start := time.Now()
	if _, err := FileHashes(ctx, server.Client(), server.URL+"/", "demo", "1.0"); err == nil {
		t.Fatal("a failing index was accepted")
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Errorf("retrying ignored the cancelled context for %v", elapsed)
	}
}
