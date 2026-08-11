package tui

import (
	"testing"

	"github.com/vaibdix/django-cli/internal/config"
	"github.com/vaibdix/django-cli/internal/scaffold"
)

func TestDeclaredTotalMatchesTheStepPlan(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		cfg  config.Project
	}{
		{name: "minimal", cfg: config.Project{}},
		{name: "tailwind", cfg: config.Project{SetupTailwind: true}},
		{name: "postgres docker", cfg: config.Project{DatabaseType: config.PostgresDocker}},
		{name: "everything", cfg: config.Project{
			CreateTemplates: true, AppName: "catalog", InitializeGit: true,
			SetupTailwind: true, SetupRestFramework: true, SetupHTMX: true,
			SetupAllAuth: true, SetupRuff: true, SetupMypy: true,
			SetupPreCommit: true, SetupMultiEnv: true, RunServer: true,
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			m := &Model{Project: tc.cfg}
			declared := m.CalculateTotalSteps()
			planned := scaffold.StepCount(tc.cfg)

			if declared != planned {
				t.Errorf("CalculateTotalSteps() = %d but the plan has %d steps; "+
					"the progress bar would not reach exactly 100%%", declared, planned)
			}
			if declared <= 0 {
				t.Fatalf("CalculateTotalSteps() = %d, want > 0 (it is a divisor)", declared)
			}
		})
	}
}

func TestEnablingTailwindAddsExactlyOneStep(t *testing.T) {
	t.Parallel()

	with := config.Project{SetupTailwind: true}
	without := config.Project{}

	delta := (&Model{Project: with}).CalculateTotalSteps() -
		(&Model{Project: without}).CalculateTotalSteps()

	if delta != 1 {
		t.Errorf("enabling Tailwind changed the step total by %d, want 1", delta)
	}
}

// Every optional feature should cost one step, so the bar advances evenly.
func TestEachFeatureAddsOneStep(t *testing.T) {
	t.Parallel()

	features := map[string]func(*config.Project){
		"templates": func(c *config.Project) { c.CreateTemplates = true },
		"git":       func(c *config.Project) { c.InitializeGit = true },
		"tailwind":  func(c *config.Project) { c.SetupTailwind = true },
		"htmx":      func(c *config.Project) { c.SetupHTMX = true },
		"allauth":   func(c *config.Project) { c.SetupAllAuth = true },
		"ruff":      func(c *config.Project) { c.SetupRuff = true },
		"mypy":      func(c *config.Project) { c.SetupMypy = true },
		"precommit": func(c *config.Project) { c.SetupPreCommit = true },
		"multienv":  func(c *config.Project) { c.SetupMultiEnv = true },
		"rest":      func(c *config.Project) { c.SetupRestFramework = true },
		"app":       func(c *config.Project) { c.AppName = "catalog" },
	}

	base := scaffold.StepCount(config.Project{})

	for name, enable := range features {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			cfg := config.Project{}
			enable(&cfg)

			if got := scaffold.StepCount(cfg) - base; got != 1 {
				t.Errorf("enabling %s changed the step total by %d, want 1", name, got)
			}
		})
	}
}

func TestAdvanceProgressClampsAtTotal(t *testing.T) {
	t.Parallel()

	m := &Model{TotalSteps: 3}

	var last float64
	for i := 0; i < 5; i++ {
		last = m.advanceProgress()
	}

	if m.CompletedSteps > m.TotalSteps {
		t.Errorf("CompletedSteps = %d, want it clamped to TotalSteps = %d",
			m.CompletedSteps, m.TotalSteps)
	}
	if last > 1.0 {
		t.Errorf("progress fraction = %v, want <= 1.0", last)
	}
}

func TestProgressReachesExactlyOneHundredPercent(t *testing.T) {
	t.Parallel()

	cfg := config.Project{SetupTailwind: true, SetupRuff: true, AppName: "catalog"}
	total := scaffold.StepCount(cfg)

	m := &Model{Project: cfg, TotalSteps: total}

	var last float64
	for i := 0; i < total; i++ {
		last = m.advanceProgress()
	}

	if last != 1.0 {
		t.Errorf("after %d of %d steps the bar reads %v, want exactly 1.0", total, total, last)
	}
}

func TestAdvanceProgressHandlesZeroTotal(t *testing.T) {
	t.Parallel()

	m := &Model{TotalSteps: 0}

	if got := m.advanceProgress(); got != 0 {
		t.Errorf("advanceProgress() with TotalSteps=0 = %v, want 0", got)
	}
}
