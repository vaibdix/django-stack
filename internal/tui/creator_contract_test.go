package tui

import (
	"testing"

	"github.com/vaibdix/django-cli/internal/config"
	"github.com/vaibdix/django-cli/internal/scaffold"
)

// Progress-accounting contract for the setup screen.
//
// The step-ordering half of this file moved to internal/scaffold/plan_test.go,
// where the plan it describes now lives.
//
// What remains was written in Phase 0 to record a defect: the declared step
// total and the number of progress updates the engine actually emitted did not
// match, and the instruction left here was "once progress is driven by the step
// plan, invert this test to assert equality." That has happened, so these now
// assert the property rather than document its absence.

// The bar's denominator and the number of updates it will receive must be the
// same number. They are now both derived from the step plan, so this holds by
// construction — the test is here to keep it that way.
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

// Tailwind used to advance the bar twice against a budget of one, because the
// engine called Progress from both the plan and the Tailwind step itself.
// Enabling a feature must now cost exactly the steps it adds.
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

// advanceProgress used to increment unconditionally, so more progress calls
// than planned steps drove the bar past 100% — it reached 200% in the minimal
// configuration. It now clamps.
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

// Running exactly the planned number of steps must land on precisely 100%.
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

// A zero TotalSteps must not produce NaN or +Inf.
func TestAdvanceProgressHandlesZeroTotal(t *testing.T) {
	t.Parallel()

	m := &Model{TotalSteps: 0}

	if got := m.advanceProgress(); got != 0 {
		t.Errorf("advanceProgress() with TotalSteps=0 = %v, want 0", got)
	}
}
