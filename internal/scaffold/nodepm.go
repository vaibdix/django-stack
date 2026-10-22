package scaffold

import (
	"context"
	"strings"

	"github.com/vaibdix/django-stack/internal/command"
	"github.com/vaibdix/django-stack/internal/pyenv"
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

var detectNodeManager = func(ctx context.Context, run command.Runner) (NodeManager, bool) {
	if pyenv.CommandWorks(ctx, run, pnpm.Name) {
		return pnpm, true
	}
	if pyenv.CommandWorks(ctx, run, NPM.Name) {
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

func (e *Env) commandVersion(name string) string {
	output, err := pyenv.ProbeVersion(e.context(), e.commandRunner(), name)
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
