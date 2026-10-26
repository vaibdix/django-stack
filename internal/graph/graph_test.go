package graph

import (
	"context"
	"errors"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
)

type log struct {
	mu    sync.Mutex
	order []string
}

func (l *log) add(name string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.order = append(l.order, name)
}

func step(name string, needs ...string) Node[*log] {
	return Node[*log]{Name: name, Needs: needs, Run: func(l *log, _ context.Context) error { l.add(name); return nil }}
}

func TestANodeRunsOnlyAfterEverythingItNeeds(t *testing.T) {
	t.Parallel()

	for range 50 {
		l := &log{}
		nodes := []Node[*log]{step("a"), step("b", "a"), step("c", "a"), step("d", "b", "c")}
		if err := Run(t.Context(), l, nodes, Hooks{}); err != nil {
			t.Fatalf("Run: %v", err)
		}
		pos := map[string]int{}
		for i, name := range l.order {
			pos[name] = i
		}
		if len(l.order) != 4 || pos["a"] > pos["b"] || pos["a"] > pos["c"] || pos["b"] > pos["d"] || pos["c"] > pos["d"] {
			t.Fatalf("order %v breaks a need", l.order)
		}
	}
}

func TestIndependentNodesOverlap(t *testing.T) {
	t.Parallel()

	release := make(chan struct{})
	both := Node[*log]{Name: "waits", Run: func(*log, context.Context) error { <-release; return nil }}
	other := Node[*log]{Name: "frees", Run: func(*log, context.Context) error { close(release); return nil }}
	done := make(chan error, 1)
	go func() { done <- Run(t.Context(), &log{}, []Node[*log]{both, other}, Hooks{}) }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("two nodes with no needs did not run at the same time")
	}
}

func TestASkippedNodeStillWaitsForItsNeeds(t *testing.T) {
	t.Parallel()

	l := &log{}
	skipped := step("skipped", "a")
	skipped.Skip = true
	nodes := []Node[*log]{step("a"), skipped, step("b", "skipped")}
	if err := Run(t.Context(), l, nodes, Hooks{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !slices.Equal(l.order, []string{"a", "b"}) {
		t.Errorf("order %v, want a then b with the skipped node in between running nothing", l.order)
	}
}

func TestTheFirstFailureIsReportedAndTheRestIsCancelled(t *testing.T) {
	t.Parallel()

	boom := errors.New("boom")
	l := &log{}
	nodes := []Node[*log]{
		{Name: "slow", Run: func(_ *log, ctx context.Context) error { <-ctx.Done(); return ctx.Err() }},
		{Name: "fails", Run: func(*log, context.Context) error { return boom }},
		step("after", "fails"),
	}
	err := Run(t.Context(), l, nodes, Hooks{})
	var failure *Failure
	if !errors.As(err, &failure) || failure.Node != "fails" || !failure.Ran || !errors.Is(err, boom) {
		t.Fatalf("Run = %v, want the failure of \"fails\" and not the cancelled sibling", err)
	}
	if slices.Contains(l.order, "after") {
		t.Error("a node ran after the node it needs failed")
	}
}

func TestACancelledRunNamesTheNodeItDidNotStart(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	err := Run(ctx, &log{}, []Node[*log]{step("first")}, Hooks{})
	var failure *Failure
	if !errors.As(err, &failure) || failure.Node != "first" || failure.Ran || !errors.Is(err, context.Canceled) {
		t.Fatalf("Run = %v, want a cancellation before \"first\" ran", err)
	}
}

func TestHooksSeeEveryNodeStartAndFinish(t *testing.T) {
	t.Parallel()

	var mu sync.Mutex
	var seen []string
	tick := time.Unix(0, 0)
	hooks := Hooks{
		Started: func(name string) { mu.Lock(); seen = append(seen, "start "+name); mu.Unlock() },
		Finished: func(name string, took time.Duration, err error) {
			mu.Lock()
			seen = append(seen, "finish "+name+" "+took.String())
			mu.Unlock()
		},
		Clock: func() time.Time { mu.Lock(); defer mu.Unlock(); tick = tick.Add(time.Second); return tick },
	}
	if err := Run(t.Context(), &log{}, []Node[*log]{step("a"), step("b", "a")}, hooks); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if want := []string{"start a", "finish a 1s", "start b", "finish b 1s"}; !slices.Equal(seen, want) {
		t.Errorf("hooks saw %v, want %v", seen, want)
	}
}

func TestANeedMustBeListedEarlier(t *testing.T) {
	t.Parallel()

	for name, nodes := range map[string][]Node[*log]{
		"forward":   {step("a", "b"), step("b")},
		"unknown":   {step("a", "nope")},
		"duplicate": {step("a"), step("a")},
		"self":      {step("a", "a")},
	} {
		if err := Run(t.Context(), &log{}, nodes, Hooks{}); err == nil || !strings.HasPrefix(err.Error(), "graph:") {
			t.Errorf("%s: Run = %v, want a graph error before anything runs", name, err)
		}
	}
}
