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
		cfg  config.Choices
	}{
		{name: "minimal", cfg: config.Choices{}},
		{name: "tailwind", cfg: config.Choices{Stack: config.StackTailwind}},
		{name: "postgres docker", cfg: config.Choices{Database: config.PostgresDocker}},
		{name: "everything", cfg: config.Choices{
			App:    "catalog",
			Stack:  config.StackTailwindHTMX,
			AddOns: config.NewSet(config.Git, config.RestFramework, config.AllAuth, config.Ruff, config.Mypy, config.PreCommit, config.MultiEnv),
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			m := &Model{choices: tc.cfg}
			declared := m.calculateTotalSteps()
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

	with := config.Choices{Stack: config.StackTailwind}
	without := config.Choices{Stack: config.StackStandard}

	delta := (&Model{choices: with}).calculateTotalSteps() -
		(&Model{choices: without}).calculateTotalSteps()

	if delta != 2 {
		t.Errorf("enabling Tailwind changed the step total by %d, want 2 "+
			"(setup tailwind and build tailwind css)", delta)
	}
}

func TestEachFeatureAddsItsBudgetedSteps(t *testing.T) {
	t.Parallel()

	features := map[string]struct {
		enable func(*config.Choices)
		want   int
	}{
		"templates": {func(c *config.Choices) { c.Stack = config.StackStandard }, 1},
		"git":       {func(c *config.Choices) { c.AddOns = config.NewSet(config.Git) }, 1},
		"tailwind":  {func(c *config.Choices) { c.Stack = config.StackTailwind }, 3},
		"htmx":      {func(c *config.Choices) { c.Stack = config.StackHTMX }, 2},
		"allauth":   {func(c *config.Choices) { c.AddOns = config.NewSet(config.AllAuth) }, 1},
		"ruff":      {func(c *config.Choices) { c.AddOns = config.NewSet(config.Ruff) }, 1},
		"mypy":      {func(c *config.Choices) { c.AddOns = config.NewSet(config.Mypy) }, 1},
		"precommit": {func(c *config.Choices) { c.AddOns = config.NewSet(config.PreCommit) }, 2},
		"multienv":  {func(c *config.Choices) { c.AddOns = config.NewSet(config.MultiEnv) }, 1},
		"rest":      {func(c *config.Choices) { c.AddOns = config.NewSet(config.RestFramework) }, 1},
		"ninja":     {func(c *config.Choices) { c.AddOns = config.NewSet(config.Ninja) }, 1},
		"app":       {func(c *config.Choices) { c.App = "catalog" }, 1},
	}

	base := scaffold.StepCount(config.Choices{Stack: config.StackAPIOnly})

	for name, tc := range features {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			cfg := config.Choices{Stack: config.StackAPIOnly}
			tc.enable(&cfg)

			if got := scaffold.StepCount(cfg) - base; got != tc.want {
				t.Errorf("enabling %s changed the step total by %d, want %d", name, got, tc.want)
			}
		})
	}
}

func TestAdvanceProgressClampsAtTotal(t *testing.T) {
	t.Parallel()

	m := &Model{totalSteps: 3}

	var last float64
	for range 5 {
		last = m.advanceProgress()
	}

	if m.completedSteps > m.totalSteps {
		t.Errorf("CompletedSteps = %d, want it clamped to TotalSteps = %d",
			m.completedSteps, m.totalSteps)
	}
	if last > 1.0 {
		t.Errorf("progress fraction = %v, want <= 1.0", last)
	}
}

func TestProgressReachesExactlyOneHundredPercent(t *testing.T) {
	t.Parallel()

	cfg := config.Choices{App: "catalog", Stack: config.StackTailwind, AddOns: config.NewSet(config.Ruff)}
	total := scaffold.StepCount(cfg)

	m := &Model{choices: cfg, totalSteps: total}

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

	m := &Model{totalSteps: 0}

	if got := m.advanceProgress(); got != 0 {
		t.Errorf("advanceProgress() with TotalSteps=0 = %v, want 0", got)
	}
}
