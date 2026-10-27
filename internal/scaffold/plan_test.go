package scaffold

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/vaibdix/django-stack/internal/config"
	"github.com/vaibdix/django-stack/internal/graph"
	"github.com/vaibdix/django-stack/internal/plan"
	"github.com/vaibdix/django-stack/internal/render"
)

func TestTheExecutorRunsEveryPlannedEffect(t *testing.T) {
	t.Parallel()

	e := &Env{Cfg: mustSpec(config.Choices{})}
	nodes, err := e.nodes()
	if err != nil {
		t.Fatalf("nodes: %v", err)
	}
	if err := graph.Validate(nodes); err != nil {
		t.Errorf("the plan is not a valid graph: %v", err)
	}
	if len(nodes) != len(plan.Effects()) {
		t.Errorf("%d nodes for %d effects", len(nodes), len(plan.Effects()))
	}
}

func TestAnEffectWritesOnlyTheFilesItDeclares(t *testing.T) {
	t.Parallel()

	e := &Env{Cfg: mustSpec(config.Choices{}), dir: t.TempDir()}
	effect := plan.Effect{ID: "write readme", Produces: []string{plan.Group(render.GroupReadme)}}
	ctx := withEffect(t.Context(), effect)
	if err := e.emitGroup(ctx, render.GroupGitignore); err == nil || !strings.Contains(err.Error(), "does not declare") {
		t.Errorf("emitGroup outside the declared groups = %v, want a refusal", err)
	}
	if err := e.emitGroup(ctx, render.GroupReadme); err != nil {
		t.Errorf("emitGroup of a declared group: %v", err)
	}
}

func TestACompensationIsRecordedOnlyAfterItsEffectSucceeds(t *testing.T) {
	t.Parallel()

	stopped := 0
	compensate := func(*Env, context.Context) error { stopped++; return nil }
	boom := errors.New("boom")
	effect := plan.Effect{ID: "start"}

	e := &Env{}
	if err := effectRunner(effect, effectRun{run: func(*Env, context.Context) error { return boom }, compensate: compensate})(e, t.Context()); !errors.Is(err, boom) {
		t.Fatalf("run = %v, want boom", err)
	}
	if len(e.undo) != 0 {
		t.Errorf("a failed effect recorded %d compensations", len(e.undo))
	}
	if err := effectRunner(effect, effectRun{run: func(*Env, context.Context) error { return nil }, compensate: compensate})(e, t.Context()); err != nil {
		t.Fatalf("run: %v", err)
	}
	if len(e.undo) != 1 {
		t.Fatalf("a successful effect recorded %d compensations, want 1", len(e.undo))
	}
	if err := e.undo[0](t.Context()); err != nil || stopped != 1 {
		t.Errorf("the recorded compensation did not call the effect's own: err=%v stopped=%d", err, stopped)
	}
}
