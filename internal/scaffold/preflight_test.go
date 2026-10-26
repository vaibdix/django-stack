package scaffold

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vaibdix/django-stack/internal/command"
	"github.com/vaibdix/django-stack/internal/command/commandtest"
	"github.com/vaibdix/django-stack/internal/config"
	"github.com/vaibdix/django-stack/internal/livetest"
	"github.com/vaibdix/django-stack/internal/port"
	"github.com/vaibdix/django-stack/internal/report"
)

func preflightEnv(t *testing.T, cfg config.Choices, fake *commandtest.Fake) *Env {
	t.Helper()
	cfg.Name, cfg.Django = "shop", "6.1.2"
	python := config.Resolved{PythonPath: filepath.Join(t.TempDir(), "python3")}
	return &Env{cfg: mustSpec(cfg), resolved: python, root: filepath.Join(t.TempDir(), "shop"), reporter: report.Nop{}, ports: port.Ports{Run: fake, Clock: reviewClock}}
}

func TestPreflightFailuresStopTheRunBeforeAnythingIsCreated(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		cfg     config.Choices
		arrange func(*commandtest.Fake)
		lookup  func(context.Context, command.Runner) (NodeManager, bool)
		want    string
	}{
		"git missing": {
			cfg:     config.Choices{Database: config.SQLite, AddOns: config.NewSet(config.Git)},
			arrange: func(f *commandtest.Fake) { f.FailWith("git --version", "not found") },
			want:    "git is not on PATH",
		},
		"docker missing": {
			cfg:     config.Choices{Database: config.PostgresDocker},
			arrange: func(f *commandtest.Fake) { f.FailWith("docker --version", "not found") },
			want:    "docker is not installed",
		},
		"compose missing": {
			cfg:     config.Choices{Database: config.MySQLDocker},
			arrange: func(f *commandtest.Fake) { f.FailWith("docker compose version", "unknown command") },
			want:    "docker compose is not available",
		},
		"docker stopped": {
			cfg:     config.Choices{Database: config.PostgresDocker},
			arrange: func(f *commandtest.Fake) { f.FailWith("docker info", "cannot connect") },
			want:    "docker is not running",
		},
		"node too old": {
			cfg:     config.Choices{Database: config.SQLite, Stack: config.StackTailwind, App: "catalog"},
			arrange: func(f *commandtest.Fake) { f.Responses["node --version"] = []byte("v18.20.4\n") },
			lookup:  func(context.Context, command.Runner) (NodeManager, bool) { return pnpm, true },
			want:    "needs Node.js 20 or newer",
		},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			fake := commandtest.NewFake()
			c.arrange(fake)
			e := preflightEnv(t, c.cfg, fake)
			e.deps.nodeManager = c.lookup
			err := e.Generate(context.Background())
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("Generate = %v, want an error mentioning %q", err, c.want)
			}
			if _, statErr := os.Stat(e.root); !os.IsNotExist(statErr) {
				t.Errorf("the project directory was created before the preflight failed")
			}
		})
	}
}

func TestACurrentNodePassesThePreflight(t *testing.T) {
	t.Parallel()

	fake := commandtest.NewFake()
	fake.Responses["node --version"] = []byte("v24.22.1\n")
	if err := requireNode(context.Background(), fake, tailwindMinNode()); err != nil {
		t.Errorf("Node 24 was refused: %v", err)
	}
}

func TestANameTheInterpreterShipsIsRefusedBeforeAnythingIsCreated(t *testing.T) {
	t.Parallel()

	livetest.Require(t, "probe a real Python 3.10")
	python, err := exec.CommandContext(t.Context(), "uv", "python", "find", "3.10").Output()
	if err != nil {
		t.Fatalf("uv python find 3.10: %v", err)
	}
	e := &Env{
		cfg: mustSpec(config.Choices{
			Name:     "telnetlib",
			Django:   "5.2.18",
			Database: config.SQLite,
		}),
		resolved: config.Resolved{PythonPath: strings.TrimSpace(string(python)), PythonVersion: "3.10.18"},
		root:     filepath.Join(t.TempDir(), "telnetlib"),
		reporter: report.Nop{},
	}
	if err := e.Generate(context.Background()); err == nil || !strings.Contains(err.Error(), "standard library") {
		t.Fatalf("Generate = %v, want the standard-library refusal", err)
	}
	if _, err := os.Stat(e.root); !os.IsNotExist(err) {
		t.Error("the project directory was created before the name was refused")
	}
}
