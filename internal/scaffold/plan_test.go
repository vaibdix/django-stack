package scaffold

import (
	"testing"

	"github.com/vaibdix/django-cli/internal/config"
)

// canonicalStepOrder was transcribed from the original straight-line
// CreateProject before the refactor, and lived in internal/tui as a
// characterisation test with the note: "Phase 3 replaces the straight-line
// function with Plan(cfg) []Step; that Plan must reproduce this order exactly."
//
// This is that check. It is the payoff for having written the order down before
// touching anything.
var canonicalStepOrder = []string{
	"create virtual environment",
	"install django",
	"create django project",
	"setup docker database", // postgres-docker / mysql-docker only
	"create .env",           // MUST come after "setup docker database"
	"create .env.example",
	"write docker-compose.yml", // docker databases only
	"write postgres readme",    // postgres-string only
	"configure django settings",
	"setup project urls",
	"initialize git",
	"setup global templates",
	"setup tailwind",
	"setup htmx",
	"setup allauth",
	"setup ruff",
	"setup mypy",
	"setup pre-commit", // MUST come after "setup ruff"
	"create django app",
	"setup rest framework",
	"setup multi-env", // MUST come last of the settings writers
	"run migrations",
	"write superuser guide",
	"run scaffold formatting",
	"report server instructions",
	"write vscode config",
	"write dev start script",
}

func TestPlanMatchesCanonicalOrder(t *testing.T) {
	t.Parallel()

	got := plan()
	if len(got) != len(canonicalStepOrder) {
		t.Fatalf("plan has %d steps, canonical order has %d", len(got), len(canonicalStepOrder))
	}

	for i, want := range canonicalStepOrder {
		if got[i].name != want {
			t.Errorf("step %d: plan has %q, canonical order has %q", i, got[i].name, want)
		}
	}
}

// The three orderings the original code called out in comments. Asserting them
// on the plan means a future reordering fails a test instead of producing a
// project that looks fine and does not run.
func TestLoadBearingOrderingsHold(t *testing.T) {
	t.Parallel()

	pos := make(map[string]int)
	for i, s := range plan() {
		if _, dup := pos[s.name]; dup {
			t.Fatalf("step %q appears twice in the plan", s.name)
		}
		pos[s.name] = i
	}

	constraints := []struct {
		before, after, why string
	}{
		{"setup docker database", "create .env",
			".env reuses the password generated for the container"},
		{"setup ruff", "setup pre-commit",
			"the pre-commit config references the ruff configuration"},
		{"setup rest framework", "setup multi-env",
			"multi-env turns settings.py into a package, so it must run after every settings writer"},
		{"configure django settings", "setup multi-env",
			"multi-env turns settings.py into a package, so it must run after every settings writer"},
		{"setup allauth", "setup multi-env",
			"multi-env turns settings.py into a package, so it must run after every settings writer"},
	}

	for _, c := range constraints {
		if pos[c.before] >= pos[c.after] {
			t.Errorf("%q must run before %q — %s (positions %d and %d)",
				c.before, c.after, c.why, pos[c.before], pos[c.after])
		}
	}
}

// plan() is pure: it selects steps and runs nothing. That is what makes the
// ordering assertable without a virtualenv, Docker or a network.
func TestPlanSelectsStepsByConfiguration(t *testing.T) {
	t.Parallel()

	names := func(cfg config.Project) []string {
		var out []string
		for _, s := range plan() {
			if s.applies(cfg) {
				out = append(out, s.name)
			}
		}
		return out
	}

	tests := []struct {
		name       string
		cfg        config.Project
		wantSteps  []string
		unwantStep []string
	}{
		{
			name: "minimal sqlite project",
			cfg:  config.Project{ProjectName: "demo", DatabaseType: config.SQLite},
			wantSteps: []string{
				"create virtual environment", "install django", "create .env", "run migrations",
			},
			unwantStep: []string{
				"setup docker database", "write docker-compose.yml", "write postgres readme",
				"setup tailwind", "setup rest framework", "setup multi-env", "create django app",
			},
		},
		{
			name: "postgres in docker",
			cfg:  config.Project{ProjectName: "demo", DatabaseType: config.PostgresDocker},
			wantSteps: []string{
				"setup docker database", "write docker-compose.yml",
			},
			unwantStep: []string{"write postgres readme"},
		},
		{
			name: "external postgres gets a readme, not a container",
			cfg:  config.Project{ProjectName: "demo", DatabaseType: config.PostgresString},
			wantSteps: []string{
				"write postgres readme",
			},
			unwantStep: []string{"setup docker database", "write docker-compose.yml"},
		},
		{
			name: "everything on",
			cfg: config.Project{
				ProjectName: "demo", AppName: "catalog", DatabaseType: config.MySQLDocker,
				CreateTemplates: true, InitializeGit: true, SetupTailwind: true,
				SetupRestFramework: true, SetupHTMX: true, SetupAllAuth: true,
				SetupRuff: true, SetupMypy: true, SetupPreCommit: true,
				SetupMultiEnv: true, RunServer: true,
			},
			wantSteps: []string{
				"setup docker database", "setup tailwind", "setup htmx", "setup allauth",
				"setup ruff", "setup mypy", "setup pre-commit", "create django app",
				"setup rest framework", "setup multi-env", "report server instructions",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			selected := names(tc.cfg)
			has := make(map[string]bool, len(selected))
			for _, n := range selected {
				has[n] = true
			}

			for _, want := range tc.wantSteps {
				if !has[want] {
					t.Errorf("step %q should run for this configuration but does not", want)
				}
			}
			for _, unwanted := range tc.unwantStep {
				if has[unwanted] {
					t.Errorf("step %q should not run for this configuration but does", unwanted)
				}
			}
		})
	}
}

// Every step must be runnable and named; a nil run would panic mid-scaffold.
func TestEveryStepIsWellFormed(t *testing.T) {
	t.Parallel()

	for i, s := range plan() {
		if s.name == "" {
			t.Errorf("step %d has no name", i)
		}
		if s.run == nil {
			t.Errorf("step %q has a nil run function", s.name)
		}
	}
}
