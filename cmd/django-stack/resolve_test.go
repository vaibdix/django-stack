package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/vaibdix/django-stack/internal/command"
	"github.com/vaibdix/django-stack/internal/pyenv"
)

func TestAReleaseTheIndexDoesNotServeFailsBeforeAnyProjectExists(t *testing.T) {
	options, _ := pyenv.Discover(context.Background(), command.NewOS())
	python, ok := newestPatch(options, 13)
	if !ok {
		python, ok = newestPatch(options, 12)
	}
	if !ok {
		t.Skip("needs a Python 3.12 or 3.13 interpreter")
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/simple/django/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/vnd.pypi.simple.v1+json")
		_, _ = io.WriteString(w, `{"meta": {"api-version": "1.4"}, "name": "django", "versions": ["6.1.2"],
			"files": [{"filename": "django-6.1.2-py3-none-any.whl", "requires-python": ">=3.12"}]}`)
	}))
	t.Cleanup(server.Close)
	t.Setenv("UV_DEFAULT_INDEX", server.URL+"/simple/")
	t.Setenv("PIP_INDEX_URL", server.URL+"/simple/")

	dir := t.TempDir()
	t.Chdir(dir)
	answers := filepath.Join(dir, "answers.json")
	content := fmt.Sprintf(`{"name": "missing-release", "python": %q, "django": "6.1.99", "database": "sqlite", "stack": "standard", "app": "catalog"}`, python.Path)
	if err := os.WriteFile(answers, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	err := run([]string{"-config", answers}, io.Discard, io.Discard)
	if code := exitCode(err); code != exitFailed {
		t.Fatalf("exit status %d (%v), want %d", code, err, exitFailed)
	}
	if _, statErr := os.Stat(filepath.Join(dir, "missing-release")); !os.IsNotExist(statErr) {
		t.Errorf("a project directory exists after the release was rejected: %v", statErr)
	}
}
