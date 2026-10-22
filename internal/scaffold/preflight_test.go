package scaffold

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vaibdix/django-stack/internal/command/commandtest"
	"github.com/vaibdix/django-stack/internal/config"
)

func preflightEnv(t *testing.T, cfg config.Project, fake *commandtest.Fake) *Env {
	t.Helper()
	cfg.ProjectName, cfg.DjangoVersion = "shop", "6.1.2"
	cfg.SelectedPythonPath = filepath.Join(t.TempDir(), "python3")
	return &Env{Cfg: cfg, Root: filepath.Join(t.TempDir(), "shop"), Runner: fake, Reporter: NopReporter{}, Clock: reviewClock}
}

func TestPreflightFailuresStopTheRunBeforeAnythingIsCreated(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		cfg     config.Project
		arrange func(*commandtest.Fake)
		lookup  func() (NodeManager, bool)
		want    string
	}{
		"git missing": {
			cfg:     config.Project{InitializeGit: true, DatabaseType: config.SQLite},
			arrange: func(f *commandtest.Fake) { f.FailWith("git --version", "not found") },
			want:    "git is not on PATH",
		},
		"docker missing": {
			cfg:     config.Project{DatabaseType: config.PostgresDocker},
			arrange: func(f *commandtest.Fake) { f.FailWith("docker --version", "not found") },
			want:    "docker is not installed",
		},
		"compose missing": {
			cfg:     config.Project{DatabaseType: config.MySQLDocker},
			arrange: func(f *commandtest.Fake) { f.FailWith("docker compose version", "unknown command") },
			want:    "docker compose is not available",
		},
		"docker stopped": {
			cfg:     config.Project{DatabaseType: config.PostgresDocker},
			arrange: func(f *commandtest.Fake) { f.FailWith("docker info", "cannot connect") },
			want:    "docker is not running",
		},
		"node too old": {
			cfg:     config.Project{SetupTailwind: true, DatabaseType: config.SQLite},
			arrange: func(f *commandtest.Fake) { f.Responses["node --version"] = []byte("v18.20.4\n") },
			lookup:  func() (NodeManager, bool) { return PNPM, true },
			want:    "needs Node.js 20 or newer",
		},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			fake := commandtest.NewFake()
			c.arrange(fake)
			e := preflightEnv(t, c.cfg, fake)
			e.nodeManagerLookup = c.lookup
			err := e.Generate(context.Background())
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("Generate = %v, want an error mentioning %q", err, c.want)
			}
			if _, statErr := os.Stat(e.Root); !os.IsNotExist(statErr) {
				t.Errorf("the project directory was created before the preflight failed")
			}
		})
	}
}

func TestACurrentNodePassesThePreflight(t *testing.T) {
	t.Parallel()

	fake := commandtest.NewFake()
	fake.Responses["node --version"] = []byte("v24.22.1\n")
	if err := requireNode(context.Background(), fake); err != nil {
		t.Errorf("Node 24 was refused: %v", err)
	}
}

func TestANameTheInterpreterShipsIsRefusedBeforeAnythingIsCreated(t *testing.T) {
	if os.Getenv("DJANGO_STACK_E2E") == "" {
		t.Skip("set DJANGO_STACK_E2E=1 to probe a real Python 3.10")
	}
	python, err := exec.CommandContext(t.Context(), "uv", "python", "find", "3.10").Output()
	if err != nil {
		t.Fatalf("uv python find 3.10: %v", err)
	}
	e := &Env{
		Cfg: config.Project{
			ProjectName: "telnetlib", DjangoVersion: "5.2.18", DatabaseType: config.SQLite,
			SelectedPythonPath: strings.TrimSpace(string(python)), SelectedPythonVersion: "3.10.18",
		},
		Root:     filepath.Join(t.TempDir(), "telnetlib"),
		Reporter: NopReporter{},
	}
	if err := e.Generate(context.Background()); err == nil || !strings.Contains(err.Error(), "standard library") {
		t.Fatalf("Generate = %v, want the standard-library refusal", err)
	}
	if _, err := os.Stat(e.Root); !os.IsNotExist(err) {
		t.Error("the project directory was created before the name was refused")
	}
}
