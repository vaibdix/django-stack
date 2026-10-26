package scaffold

import (
	"context"

	"github.com/vaibdix/django-stack/internal/command"
)

type NodeManager struct {
	Name string

	installVerb string
}

var pnpm = NodeManager{
	Name:        "pnpm",
	installVerb: "add",
}

var NPM = NodeManager{
	Name:        "npm",
	installVerb: "install",
}

func NodeManagerNamed(name string) (NodeManager, bool) {
	for _, m := range []NodeManager{pnpm, NPM} {
		if m.Name == name {
			return m, true
		}
	}
	return NodeManager{}, false
}

func (d deps) detectNodeManager(ctx context.Context, run command.Runner) (NodeManager, bool) {
	if d.host().CommandWorks(ctx, run, pnpm.Name) {
		return pnpm, true
	}
	if d.host().CommandWorks(ctx, run, NPM.Name) {
		return NPM, true
	}
	return NodeManager{}, false
}

func (m NodeManager) install(packages ...string) (string, []string) {
	args := make([]string, 0, 3+len(packages))
	args = append(args, m.installVerb, "--ignore-scripts", "--save-exact")
	return m.Name, append(args, packages...)
}

func (m NodeManager) run(script string) (string, []string) {
	return m.Name, []string{"run", script}
}

func (m NodeManager) RunCommand(script string) string {
	return m.Name + " run " + script
}
