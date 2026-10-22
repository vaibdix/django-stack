package tui

import (
	"testing"

	"github.com/vaibdix/django-stack/internal/config"
	"github.com/vaibdix/django-stack/internal/scaffold"
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
			SetupPreCommit: true, SetupMultiEnv: true,
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

func TestEnablingTailwindAddsItsInstallAndItsBuild(t *testing.T) {
	t.Parallel()

	with := config.Project{SetupTailwind: true}
	without := config.Project{}

	delta := (&Model{Project: with}).CalculateTotalSteps() -
		(&Model{Project: without}).CalculateTotalSteps()

	if delta != 2 {
		t.Errorf("enabling Tailwind changed the step total by %d, want 2 "+
			"(setup tailwind and build tailwind css)", delta)
	}
}

func TestEachFeatureAddsItsBudgetedSteps(t *testing.T) {
	t.Parallel()

	features := map[string]struct {
		enable func(*config.Project)
		want   int
	}{
		"templates": {func(c *config.Project) { c.CreateTemplates = true }, 1},
		"git":       {func(c *config.Project) { c.InitializeGit = true }, 1},
		"tailwind":  {func(c *config.Project) { c.SetupTailwind = true }, 2},
		"htmx":      {func(c *config.Project) { c.SetupHTMX = true }, 1},
		"allauth":   {func(c *config.Project) { c.SetupAllAuth = true }, 1},
		"ruff":      {func(c *config.Project) { c.SetupRuff = true }, 1},
		"mypy":      {func(c *config.Project) { c.SetupMypy = true }, 1},
		"precommit": {func(c *config.Project) { c.SetupPreCommit = true }, 1},
		"multienv":  {func(c *config.Project) { c.SetupMultiEnv = true }, 1},
		"rest":      {func(c *config.Project) { c.SetupRestFramework = true }, 1},
		"ninja":     {func(c *config.Project) { c.SetupNinja = true }, 1},
		"app":       {func(c *config.Project) { c.AppName = "catalog" }, 1},
	}

	base := scaffold.StepCount(config.Project{})

	for name, tc := range features {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			cfg := config.Project{}
			tc.enable(&cfg)

			if got := scaffold.StepCount(cfg) - base; got != tc.want {
				t.Errorf("enabling %s changed the step total by %d, want %d", name, got, tc.want)
			}
		})
	}
}

func TestAdvanceProgressClampsAtTotal(t *testing.T) {
	t.Parallel()

	m := &Model{TotalSteps: 3}

	var last float64
	for range 5 {
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
	for range total {
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
