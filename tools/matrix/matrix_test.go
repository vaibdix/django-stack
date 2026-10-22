package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/vaibdix/django-stack/internal/config"
)

func TestMaskHidesValuesThatDifferOnEveryRun(t *testing.T) {
	t.Parallel()

	a := []byte("SECRET_KEY=abc\nDB_PASSWORD=one\nDB_PORT=5432\n\"createdAt\": \"2026-10-22T08:00:00Z\"\nDEBUG=True\n")
	b := []byte("SECRET_KEY=xyz\nDB_PASSWORD=two\nDB_PORT=5433\n\"createdAt\": \"2026-11-01T09:30:00Z\"\nDEBUG=True\n")
	if string(mask(a)) != string(mask(b)) {
		t.Fatalf("masked outputs differ:\n%s\n%s", mask(a), mask(b))
	}
	if c := []byte("DEBUG=False\n"); string(mask(c)) == string(mask([]byte("DEBUG=True\n"))) {
		t.Fatal("a real change was masked")
	}
}

func TestDiffProjectReportsChangedAddedAndRemovedFiles(t *testing.T) {
	t.Parallel()

	base, next := t.TempDir(), t.TempDir()
	write := func(root, rel, content string) {
		t.Helper()
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(base, "same.py", "x = 1\n")
	write(next, "same.py", "x = 1\n")
	write(base, ".env", "SECRET_KEY=a\n")
	write(next, ".env", "SECRET_KEY=b\n")
	write(base, "changed.py", "x = 1\n")
	write(next, "changed.py", "x = 2\n")
	write(base, "gone.py", "")
	write(next, "new.py", "")
	write(next, ".venv/lib/site.py", "ignored")

	res, patch := diffProject(t.Context(), base, next)
	if !slices.Equal(res.Changed, []string{"changed.py"}) || !slices.Equal(res.OnlyBaseline, []string{"gone.py"}) || !slices.Equal(res.OnlyRun, []string{"new.py"}) {
		t.Fatalf("unexpected diff: %+v", res)
	}
	if len(patch) == 0 {
		t.Fatal("no patch for the changed file")
	}
}

func TestShimForHidesOnlyTheToolsTheProjectMustNotSee(t *testing.T) {
	t.Parallel()

	cases := []struct {
		p    project
		want string
	}{
		{project{Toolchain: "uv", Node: "pnpm", Answers: config.Answers{Stack: config.StackTailwind}}, ""},
		{project{Toolchain: "pip", Node: "pnpm", Answers: config.Answers{Stack: config.StackStandard}}, "nouv"},
		{project{Toolchain: "uv", Node: "npm", Answers: config.Answers{Stack: config.StackStandard}}, ""},
		{project{Toolchain: "pip", Node: "npm", Answers: config.Answers{Stack: config.StackTailwindHTMX}}, "nouv-nopnpm"},
	}
	for _, c := range cases {
		if got := shimFor(c.p); got != c.want {
			t.Errorf("shimFor(%s/%s/%s) = %q, want %q", c.p.Toolchain, c.p.Node, c.p.Answers.Stack, got, c.want)
		}
	}
}

func TestRunOutputInsideTheRepositoryIsRefused(t *testing.T) {
	t.Parallel()

	if err := refuseInsideRepository(filepath.Join("not", "yet", "created")); err == nil {
		t.Fatal("a folder inside the repository was accepted")
	}
	if err := refuseInsideRepository(t.TempDir()); err != nil {
		t.Fatalf("a folder outside any repository was refused: %v", err)
	}
}

func TestTheDateIndexHidesFilesUploadedAfterTheCutoff(t *testing.T) {
	t.Parallel()

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/simple/djlint/" {
			http.NotFound(w, r)
			return
		}
		_, _ = io.WriteString(w, `{"meta": {"api-version": "1.4"}, "name": "djlint", "versions": ["1.46.6", "1.47.0"], "files": [
			{"filename": "djlint-1.46.6-py3-none-any.whl", "upload-time": "2026-09-01T10:00:00.000001Z"},
			{"filename": "djlint-1.47.0-py3-none-any.whl", "upload-time": "2026-10-22T14:56:20.189900Z"}]}`)
	}))
	t.Cleanup(upstream.Close)

	index, stop, err := startDateIndex(t.Context(), upstream.URL+"/simple/", time.Date(2026, 10, 22, 9, 19, 59, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(stop)

	get := func(name string) (int, string, string) {
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, index+name+"/", nil)
		if err != nil {
			t.Fatal(err)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = resp.Body.Close() }()
		body, _ := io.ReadAll(resp.Body)
		return resp.StatusCode, resp.Header.Get("Content-Type"), string(body)
	}
	if _, kind, body := get("djlint"); kind != simpleJSON || !strings.Contains(body, "djlint-1.46.6") || strings.Contains(body, "djlint-1.47.0") {
		t.Errorf("content type %q, body %s; want only the file uploaded before the cutoff", kind, body)
	}
	if status, _, _ := get("nope"); status != http.StatusNotFound {
		t.Errorf("an unknown project gave %d; want the upstream 404", status)
	}
}

func TestCleanRemovesBuildOutputButKeepsTheSourceTree(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	for _, dir := range []string{".venv/bin", "staticfiles/drf", "catalog/__pycache__", ".git/objects", "catalog/templates", ".pytest_cache"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	removeBuildOutput(root)
	for dir, want := range map[string]bool{".venv": false, "staticfiles": false, "catalog/__pycache__": false, ".pytest_cache": false, ".git": true, "catalog/templates": true} {
		if _, err := os.Stat(filepath.Join(root, dir)); (err == nil) != want {
			t.Errorf("%s exists=%t, want %t", dir, err == nil, want)
		}
	}
}
