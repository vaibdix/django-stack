package graph

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type Node[T any] struct {
	Name  string
	Needs []string
	Skip  bool
	Run   func(T, context.Context) error
}

type Hooks struct {
	Started  func(name string)
	Finished func(name string, took time.Duration, err error)
	Clock    func() time.Time
}

type Failure struct {
	Node string
	Ran  bool
	Err  error
}

func (f *Failure) Error() string { return f.Node + ": " + f.Err.Error() }

func (f *Failure) Unwrap() error { return f.Err }

func Validate[T any](nodes []Node[T]) error {
	seen := map[string]bool{}
	for _, n := range nodes {
		if n.Name == "" || seen[n.Name] {
			return fmt.Errorf("graph: node %q is unnamed or listed twice", n.Name)
		}
		for _, need := range n.Needs {
			if !seen[need] {
				return fmt.Errorf("graph: %q needs %q, which is not listed before it", n.Name, need)
			}
		}
		seen[n.Name] = true
	}
	return nil
}

func Run[T any](ctx context.Context, target T, nodes []Node[T], hooks Hooks) error {
	if err := Validate(nodes); err != nil {
		return err
	}
	hooks = hooks.withDefaults()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	index := make(map[string]int, len(nodes))
	done := make([]chan struct{}, len(nodes))
	succeeded := make([]bool, len(nodes))
	for i, n := range nodes {
		index[n.Name] = i
		done[i] = make(chan struct{})
	}

	var mu sync.Mutex
	var first *Failure
	fail := func(f *Failure) {
		mu.Lock()
		defer mu.Unlock()
		if first == nil {
			first = f
		}
		cancel()
	}

	var wg sync.WaitGroup
	for i, n := range nodes {
		wg.Go(func() {
			defer close(done[i])
			for _, need := range n.Needs {
				<-done[index[need]]
				if !succeeded[index[need]] {
					return
				}
			}
			if n.Skip {
				succeeded[i] = true
				return
			}
			if err := ctx.Err(); err != nil {
				fail(&Failure{Node: n.Name, Err: err})
				return
			}
			hooks.Started(n.Name)
			start := hooks.Clock()
			err := n.Run(target, ctx)
			hooks.Finished(n.Name, hooks.Clock().Sub(start), err)
			if err != nil {
				fail(&Failure{Node: n.Name, Ran: true, Err: err})
				return
			}
			succeeded[i] = true
		})
	}
	wg.Wait()
	if first != nil {
		return first
	}
	return nil
}

func (h Hooks) withDefaults() Hooks {
	if h.Started == nil {
		h.Started = func(string) {}
	}
	if h.Finished == nil {
		h.Finished = func(string, time.Duration, error) {}
	}
	if h.Clock == nil {
		h.Clock = time.Now
	}
	return h
}
