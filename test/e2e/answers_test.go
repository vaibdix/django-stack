package e2e

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func writeAnswers(t *testing.T, json string) (dir, path string) {
	t.Helper()
	dir = t.TempDir()
	path = filepath.Join(dir, "answers.json")
	if err := os.WriteFile(path, []byte(json), 0o644); err != nil {
		t.Fatalf("writing answers: %v", err)
	}
	return dir, path
}

func TestABadAnswersFileExitsOneAndSaysWhy(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		json string
		want string
	}{
		"typo in a key": {`{"name": "shop", "pyhton": "python3"}`, `unknown field "pyhton"`},
		"unusable python": {`{"name": "shop", "python": "/nonexistent/python3", "django": "6.1.2",
			"database": "sqlite", "stack": "api-only"}`, "not a usable Python"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			dir, path := writeAnswers(t, tc.json)

			stdout, stderr, code := runIn(t, dir, "-config", path)
			if code != 1 {
				t.Errorf("exit code = %d, want 1", code)
			}
			if !strings.Contains(stderr, tc.want) || strings.TrimSpace(stdout) != "" {
				t.Errorf("stderr does not explain the failure (want %q), or stdout was used:\nstdout: %s\nstderr: %s",
					tc.want, stdout, stderr)
			}
			if _, err := os.Stat(filepath.Join(dir, "shop")); !os.IsNotExist(err) {
				t.Errorf("a rejected answers file still created the project directory (stat: %v)", err)
			}
		})
	}
}

func TestAMissingAnswersFileExitsOne(t *testing.T) {
	t.Parallel()

	_, stderr, code := run(t, "-config", filepath.Join(t.TempDir(), "absent.json"))
	if code != 1 || !strings.Contains(stderr, "absent.json") {
		t.Errorf("exit code %d, stderr %q: want 1 and the missing path named", code, stderr)
	}
}

// TestAnswersFileGeneratesAProjectThatPassesItsOwnChecks drives a real run: a
// venv, a package install from PyPI, startproject, migrations and the
// project's own check and test suite. It needs the network and uv, so it only
// runs when DJANGO_STACK_E2E is set.
func TestAnswersFileGeneratesAProjectThatPassesItsOwnChecks(t *testing.T) {
	if os.Getenv("DJANGO_STACK_E2E") == "" {
		t.Skip("set DJANGO_STACK_E2E=1 to generate a real project")
	}
	python, err := exec.Command("uv", "python", "find", "3.13").Output()
	if err != nil {
		t.Fatalf("uv python find 3.13: %v", err)
	}

	dir, path := writeAnswers(t, `{"name": "shop", "python": "`+strings.TrimSpace(string(python))+`",
		"django": "`+e2eDjango()+`", "database": "sqlite", "stack": "htmx", "app": "catalog",
		"addons": ["rest-framework", "ruff", "mypy"]}`)

	_, stderr, code := runIn(t, dir, "-config", path)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0\n%s", code, stderr)
	}
	if !strings.Contains(stderr, "Verified the project") {
		t.Errorf("the run never reported its own verification:\n%s", stderr)
	}
	for _, file := range []string{"manage.py", "pyproject.toml", "uv.lock", "django-stack.jsonc"} {
		if _, err := os.Stat(filepath.Join(dir, "shop", file)); err != nil {
			t.Errorf("generated project has no %s: %v", file, err)
		}
	}
}

func e2eDjango() string {
	if v := os.Getenv("DJANGO_STACK_E2E_DJANGO"); v != "" {
		return v
	}
	return "6.1.2"
}
