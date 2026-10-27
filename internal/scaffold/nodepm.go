package scaffold

import (
	"context"

	"github.com/vaibdix/django-stack/internal/command"
	"github.com/vaibdix/django-stack/internal/render"
)

func (d deps) detectNodeManager(ctx context.Context, run command.Runner) (render.NodeManager, bool) {
	if d.host().CommandWorks(ctx, run, render.Pnpm.Name) {
		return render.Pnpm, true
	}
	if d.host().CommandWorks(ctx, run, render.NPM.Name) {
		return render.NPM, true
	}
	return render.NodeManager{}, false
}
