package scaffold

import (
	"strings"
	"testing"

	"github.com/vaibdix/django-stack/internal/config"
)

var canonicalStepOrder = []string{
	"create virtual environment",
	"install django",
	"setup tailwind",
	"create django project",
	"choose database credentials",
	"create .env",
	"create .env.example",
	"write docker-compose.yml",
	"setup docker database",
	"configure django settings",
	"setup project urls",
	"write .gitignore",
	"initialize git",
	"setup global templates",
	"setup allauth",
	"create shared apps",
	"create django app",
	"setup rest framework",
	"setup django ninja",
	"write dependency manifest",
	"setup pre-commit",
	"run migrations",
	"build tailwind css",
	"write readme",
	"write vscode config",
	"write dev start script",
	"write stack manifest",
	"verify project",
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
		{"choose database credentials", "create .env",
			".env carries the generated database credentials"},
		{"create .env", "setup docker database",
			"docker compose reads the credentials and port from .env"},
		{"write docker-compose.yml", "setup docker database",
			"the database is started from the compose file the project keeps"},
		{"setup docker database", "run migrations",
			"migrations need the database to be healthy"},
		{"create shared apps", "run migrations",
			"AUTH_USER_MODEL must be set before the first migration is applied"},

		{"setup global templates", "build tailwind css",
			"the stylesheet must be compiled after the templates it scans"},
		{"setup allauth", "build tailwind css",
			"the allauth pages must exist before the stylesheet is compiled"},
		{"create django app", "build tailwind css",
			"the app templates must exist before the stylesheet is compiled"},
		{"setup rest framework", "build tailwind css",
			"the REST example templates must exist before the stylesheet is compiled"},

		{"install django", "write dependency manifest",
			"the manifest records what was installed, so every installer must run first"},
		{"setup rest framework", "write dependency manifest",
			"the manifest records what was installed, so every installer must run first"},
		{"setup allauth", "write dependency manifest",
			"the manifest records what was installed, so every installer must run first"},
		{"write dependency manifest", "setup pre-commit",
			"the pip path pins each hook's tools to the versions the freeze recorded"},
		{"write dependency manifest", "run migrations",
			"a project that cannot be reinstalled should fail before it grows a database"},
		{"setup tailwind", "write stack manifest",
			"the manifest reads the node dependencies out of package.json"},
	}

	for _, c := range constraints {
		if pos[c.before] >= pos[c.after] {
			t.Errorf("%q must run before %q — %s (positions %d and %d)",
				c.before, c.after, c.why, pos[c.before], pos[c.after])
		}
	}
}

func TestPlanSelectsStepsByConfiguration(t *testing.T) {
	t.Parallel()

	names := func(cfg config.Choices) []string {
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
		cfg        config.Choices
		wantSteps  []string
		unwantStep []string
	}{
		{
			name: "minimal sqlite project",
			cfg:  config.Choices{Name: "demo", Database: config.SQLite},
			wantSteps: []string{
				"create virtual environment", "install django", "create .env", "run migrations",
			},
			unwantStep: []string{
				"setup docker database", "write docker-compose.yml", "write postgres readme",
				"setup tailwind", "setup rest framework", "create django app",
			},
		},
		{
			name: "postgres in docker",
			cfg:  config.Choices{Name: "demo", Database: config.PostgresDocker},
			wantSteps: []string{
				"setup docker database", "write docker-compose.yml",
			},
		},
		{
			name: "everything on",
			cfg: config.Choices{
				Name:     "demo",
				App:      "catalog",
				Database: config.MySQLDocker,
				Stack:    config.StackTailwindHTMX,
				AddOns:   config.NewSet(config.Git, config.RestFramework, config.AllAuth, config.Ruff, config.Mypy, config.PreCommit, config.MultiEnv),
			},
			wantSteps: []string{
				"setup docker database", "setup tailwind", "setup allauth",
				"setup pre-commit", "create django app", "setup rest framework",
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

func TestEveryProjectGetsAGitignoreWhetherOrNotGitWasPicked(t *testing.T) {
	t.Parallel()

	for _, git := range []bool{false, true} {
		found := false
		for _, s := range plan() {
			found = found || (s.name == "write .gitignore" && s.applies(config.Choices{AddOns: addOnsWhere(map[config.AddOn]bool{config.Git: git})}))
		}
		if !found {
			t.Errorf("git=%v: no .gitignore is written, so .env and .venv are one `git add .` from being committed", git)
		}
	}

	content := renderStatic("gitignore.tmpl")
	for _, entry := range []string{".env\n", ".venv/", "*.sqlite3", "node_modules/"} {
		if !strings.Contains(content, entry) {
			t.Errorf(".gitignore does not cover %q", entry)
		}
	}
}
