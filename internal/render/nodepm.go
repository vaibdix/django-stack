package render

type NodeManager struct {
	Name string

	installVerb string
}

var Pnpm = NodeManager{
	Name:        "pnpm",
	installVerb: "add",
}

var NPM = NodeManager{
	Name:        "npm",
	installVerb: "install",
}

func NodeManagerNamed(name string) (NodeManager, bool) {
	for _, m := range []NodeManager{Pnpm, NPM} {
		if m.Name == name {
			return m, true
		}
	}
	return NodeManager{}, false
}

func (m NodeManager) Install(packages ...string) (string, []string) {
	args := make([]string, 0, 3+len(packages))
	args = append(args, m.installVerb, "--ignore-scripts", "--save-exact")
	return m.Name, append(args, packages...)
}

func (m NodeManager) Run(script string) (string, []string) {
	return m.Name, []string{"run", script}
}

func (m NodeManager) RunCommand(script string) string {
	return m.Name + " run " + script
}
