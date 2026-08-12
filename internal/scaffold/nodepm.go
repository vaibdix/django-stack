package scaffold

import (
	"strings"

	"github.com/vaibdix/django-cli/internal/scaffold/pyenv"
)

type NodeManager struct {
	Name string

	installVerb string
	dlx         []string
}

var pnpmManager = NodeManager{
	Name:        "pnpm",
	installVerb: "add",
	dlx:         []string{"pnpm", "dlx"},
}

var npmManager = NodeManager{
	Name:        "npm",
	installVerb: "install",
	dlx:         []string{"npx"},
}

var detectNodeManager = func() (NodeManager, bool) {
	if pyenv.CommandWorks(pnpmManager.Name) {
		return pnpmManager, true
	}
	if pyenv.CommandWorks(npmManager.Name) {
		return npmManager, true
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

func (m NodeManager) Dlx(command string, args ...string) (string, []string) {
	full := append([]string{}, m.dlx[1:]...)
	full = append(full, command)
	return m.dlx[0], append(full, args...)
}

func (m NodeManager) DlxCommand(command string) string {
	name, args := m.Dlx(command)
	for _, arg := range args {
		name += " " + arg
	}
	return name
}

func nodeManagerOrNpm() NodeManager {
	if manager, ok := detectNodeManager(); ok {
		return manager
	}
	return npmManager
}

func NodeManagerName() string {
	return nodeManagerOrNpm().Name
}

func WatchCSSCommandFor(name string) string {
	manager := npmManager
	if name == pnpmManager.Name {
		manager = pnpmManager
	}
	return manager.RunCommand("watch:css")
}

func commandVersion(name string) string {
	output, err := pyenv.ProbeVersion(name)
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
