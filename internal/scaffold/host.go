package scaffold

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/vaibdix/django-stack/internal/catalog"
	"github.com/vaibdix/django-stack/internal/command"
)

type Host map[string]error

func ProbeHost(ctx context.Context, run command.Runner) Host {
	return probeHost(ctx, run, deps{}.withDefaults())
}

func probeHost(ctx context.Context, run command.Runner, d deps) Host {
	host := Host{}
	for _, rule := range catalog.Load().Host {
		for _, text := range rule.Needs {
			need, err := catalog.ParseNeed(text)
			if err != nil || need.Text == "docker" {
				continue
			}
			if _, done := host[need.Text]; !done {
				host[need.Text] = probeNeed(ctx, run, d, need)
			}
		}
	}
	return host
}

func probeNeed(ctx context.Context, run command.Runner, d deps, need catalog.Need) error {
	if need.Minimum > 0 {
		if requireNode(ctx, run, need.Minimum) != nil {
			return fmt.Errorf("needs Node.js %d+", need.Minimum)
		}
		return nil
	}
	for _, tool := range need.Tools {
		if d.host().CommandWorks(ctx, run, tool) {
			return nil
		}
	}
	return errors.New("needs " + strings.Join(need.Tools, " or "))
}

func (h Host) Missing(needs []catalog.Need) error {
	var missing []error
	for _, need := range needs {
		if err := h[need.Text]; err != nil {
			missing = append(missing, err)
		}
	}
	return errors.Join(missing...)
}
