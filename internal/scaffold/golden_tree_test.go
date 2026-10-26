package scaffold

import (
	"fmt"
	"strings"
	"testing"

	"github.com/vaibdix/django-stack/internal/config"
	"github.com/vaibdix/django-stack/internal/golden"
	"github.com/vaibdix/django-stack/internal/pyenv"
)

type treeCase struct {
	cfg       config.Choices
	toolchain pyenv.Toolchain
	node      NodeManager
}

func treeCases() map[string]treeCase {
	cases := map[string]treeCase{}
	for _, stack := range config.AllStacks() {
		cases["stack-"+string(stack)] = treeCase{
			cfg:       config.Choices{Name: "my-shop", Django: "6.1.2", Database: config.SQLite, Stack: stack, App: "catalog"},
			toolchain: pyenv.UV,
			node:      NPM,
		}
	}
	for _, db := range []config.Database{config.PostgresDocker, config.MySQLDocker} {
		cases["database-"+string(db)] = treeCase{
			cfg:       config.Choices{Name: "my-shop", Django: "6.1.2", Database: db, Stack: config.StackStandard, App: "catalog"},
			toolchain: pyenv.UV,
		}
	}
	cases["pip"] = treeCase{
		cfg:       config.Choices{Name: "my-shop", Django: "5.2.8", Database: config.SQLite, Stack: config.StackTailwindHTMX, App: "catalog"},
		toolchain: pyenv.Pip,
		node:      pnpm,
	}
	cases["everything-drf"] = treeCase{
		cfg:       config.Choices{Name: "my-shop", Django: "6.1.2", Database: config.PostgresDocker, Stack: config.StackTailwindHTMX, AddOns: everyAddOnBut(config.Ninja), App: "catalog"},
		toolchain: pyenv.UV,
		node:      pnpm,
	}
	cases["everything-ninja"] = treeCase{
		cfg:       config.Choices{Name: "my-shop", Django: "6.0.4", Database: config.MySQLDocker, Stack: config.StackHTMX, AddOns: everyAddOnBut(config.RestFramework), App: "catalog"},
		toolchain: pyenv.Pip,
	}
	cases["api-only-ninja"] = treeCase{
		cfg:       config.Choices{Name: "my-shop", Django: "6.1.2", Database: config.SQLite, Stack: config.StackAPIOnly, AddOns: config.NewSet(config.Ninja, config.Pydantic), App: "catalog"},
		toolchain: pyenv.UV,
	}
	return cases
}

func renderTree(t *testing.T, c treeCase) string {
	t.Helper()
	resolved := credentialsFor(c.cfg.Database, config.DatabaseCredentials{})
	resolved.PythonVersion = "Python 3.13.2"
	resolved.SecretKey = "golden-secret-key-not-for-use"
	bp, err := Build(specOf(t, c.cfg), resolved, Tools{Toolchain: c.toolchain, NodeManager: c.node})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	var b strings.Builder
	for _, path := range bp.Paths() {
		file := bp.Files[path]
		fmt.Fprintf(&b, "==> %s (%#o)\n%s", path, file.Mode, file.Content)
		if len(file.Content) > 0 && !strings.HasSuffix(string(file.Content), "\n") {
			b.WriteString("\n")
		}
	}
	return b.String()
}

func TestGoldenProjectTrees(t *testing.T) {
	t.Parallel()

	for name, c := range treeCases() {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			golden.Assert(t, "tree/"+name+".txt", renderTree(t, c))
		})
	}
}

func TestGoldenPyprojects(t *testing.T) {
	t.Parallel()

	for name, c := range treeCases() {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			e := &Env{cfg: mustSpec(c.cfg), resolved: config.Resolved{PythonVersion: "Python 3.13.2"}, toolchain: c.toolchain, node: c.node}
			e.packages.addRuntime(e.packagePlan().runtime...)
			e.packages.addDev(e.packagePlan().dev...)
			golden.Assert(t, "pyproject/"+name+".toml", e.renderPyproject(e.packages.runtime, e.packages.dev))
		})
	}
}
