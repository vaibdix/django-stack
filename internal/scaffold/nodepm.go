package scaffold

import (
	"context"
	"strings"

	"github.com/vaibdix/django-cli/internal/scaffold/command"
	"github.com/vaibdix/django-cli/internal/scaffold/pyenv"
)

type NodeManager struct {
	Name string

	installVerb string
	dlx         []string
}

// PNPM and NPM are the two package managers a Tailwind stack can build with.
var PNPM = NodeManager{
	Name:        "pnpm",
	installVerb: "add",
	dlx:         []string{"pnpm", "dlx"},
}

var NPM = NodeManager{
	Name:        "npm",
	installVerb: "install",
	dlx:         []string{"npx"},
}

var detectNodeManager = func(ctx context.Context, run command.Runner) (NodeManager, bool) {
	if pyenv.CommandWorks(ctx, run, PNPM.Name) {
		return PNPM, true
	}
	if pyenv.CommandWorks(ctx, run, NPM.Name) {
		return NPM, true
	}
	return NodeManager{}, false
}

func (m NodeManager) Install(packages ...string) (string, []string) {
	args := []string{m.installVerb, "--ignore-scripts"}
	return m.Name, append(args, packages...)
}

func (m NodeManager) Run(script string) (string, []string) {
	return m.Name, []string{"run", script}
}

func (m NodeManager) RunCommand(script string) string {
	return m.Name + " run " + script
}

// DetectNodeManager is the package manager a Tailwind run will use: pnpm when
// it works, otherwise npm, otherwise none.
func DetectNodeManager(ctx context.Context, run command.Runner) (NodeManager, bool) {
	return detectNodeManager(ctx, run)
}

func (e *Env) commandVersion(name string) string {
	output, err := pyenv.ProbeVersion(e.context(), e.runner(), name)
	if err != nil {
		return ""
	}
	for field := range strings.SplitSeq(strings.TrimSpace(output), " ") {
		if len(field) > 0 && field[0] >= '0' && field[0] <= '9' {
			return field
		}
	}
	return ""
}
