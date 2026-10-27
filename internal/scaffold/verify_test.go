package scaffold

import (
	"errors"
	"strings"
	"testing"

	"github.com/vaibdix/django-stack/internal/command/commandtest"
	"github.com/vaibdix/django-stack/internal/config"
	"github.com/vaibdix/django-stack/internal/port"
	"github.com/vaibdix/django-stack/internal/pyenv"
)

func TestVerificationRunsTheChecksThePickedToolsSupport(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	python := pyenv.InterpreterPath(root)

	cases := []struct {
		name string
		cfg  config.Choices
		want []string
		skip []string
	}{
		{
			name: "no tools",
			cfg:  config.Choices{Name: "demo"},
			want: []string{python + " manage.py check", python + " manage.py test --noinput"},
			skip: []string{"ruff", "mypy"},
		},
		{
			name: "everything",
			cfg:  config.Choices{Name: "demo", AddOns: config.NewSet(config.Ruff, config.Mypy, config.MultiEnv)},
			want: []string{
				python + " manage.py test --noinput --settings=demo.settings.test",
				python + " -m ruff check .",
				python + " -m mypy .",
			},
		},
	}
	for _, tc := range cases {
		fake := commandtest.NewFake()
		e := &Env{Cfg: mustSpec(tc.cfg), ports: port.Ports{Run: fake}}
		e.dir = root
		if err := e.verifyProject(t.Context()); err != nil {
			t.Fatalf("%s: verifyProject: %v", tc.name, err)
		}
		for _, want := range tc.want {
			if !fake.Ran(want) {
				t.Errorf("%s: %q was not run; ran %q", tc.name, want, fake.CommandLines())
			}
		}
		for _, tool := range tc.skip {
			if strings.Contains(strings.Join(fake.CommandLines(), "\n"), tool) {
				t.Errorf("%s: ran %s although it was not picked", tc.name, tool)
			}
		}
	}
}

func TestAFailingTestSuiteFailsTheRunWithItsOutput(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	python := pyenv.InterpreterPath(root)
	fake := commandtest.NewFake()
	failing := python + " manage.py test --noinput"
	fake.Responses[failing] = []byte("FAIL: test_index_renders\nAssertionError: 301 != 200")
	fake.FailWith(failing, "exit status 1")

	e := &Env{Cfg: mustSpec(config.Choices{Name: "demo"}), ports: port.Ports{Run: fake}}
	e.dir = root
	err := e.verifyProject(t.Context())

	var verification *VerificationError
	if !errors.As(err, &verification) {
		t.Fatalf("verifyProject = %v, want a *VerificationError", err)
	}
	if verification.Check != "test suite" || !strings.Contains(err.Error(), "301 != 200") {
		t.Errorf("the error does not say which check failed and why: %v", err)
	}
	if strings.Count(err.Error(), "301 != 200") != 1 {
		t.Errorf("the output is repeated:\n%v", err)
	}
}

func TestTheFirstFailingCheckInOrderIsReported(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	python := pyenv.InterpreterPath(root)
	fake := commandtest.NewFake()
	fake.FailWith(python+" -m mypy .", "exit status 1")
	fake.FailWith(python+" -m ruff check .", "exit status 1")

	e := &Env{Cfg: mustSpec(config.Choices{Name: "demo", AddOns: config.NewSet(config.Ruff, config.Mypy)}), ports: port.Ports{Run: fake}}
	e.dir = root
	var verification *VerificationError
	if err := e.verifyProject(t.Context()); !errors.As(err, &verification) || verification.Check != "ruff check" {
		t.Fatalf("verifyProject = %v, want the ruff failure, which comes before mypy's", err)
	}
	if !fake.Ran(python + " -m mypy .") {
		t.Error("mypy did not run alongside ruff")
	}
}

func TestATestSuiteRunsOnlyAfterTheSystemCheckPasses(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	python := pyenv.InterpreterPath(root)
	fake := commandtest.NewFake()
	fake.FailWith(python+" manage.py check", "exit status 1")

	e := &Env{Cfg: mustSpec(config.Choices{Name: "demo"}), ports: port.Ports{Run: fake}}
	e.dir = root
	_ = e.verifyProject(t.Context())
	if fake.Ran(python + " manage.py test --noinput") {
		t.Error("the test suite ran although the system check failed")
	}
}
