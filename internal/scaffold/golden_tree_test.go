package scaffold

import (
	"github.com/vaibdix/django-stack/internal/config"
	"github.com/vaibdix/django-stack/internal/pyenv"
	"github.com/vaibdix/django-stack/internal/render"
)

type treeCase struct {
	cfg       config.Choices
	toolchain pyenv.Toolchain
	node      render.NodeManager
}

func treeCases() map[string]treeCase {
	cases := map[string]treeCase{}
	for _, stack := range config.AllStacks() {
		cases["stack-"+string(stack)] = treeCase{
			cfg:       config.Choices{Name: "my-shop", Django: "6.1.2", Database: config.SQLite, Stack: stack, App: "catalog"},
			toolchain: pyenv.UV,
			node:      render.NPM,
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
		node:      render.Pnpm,
	}
	cases["everything-drf"] = treeCase{
		cfg:       config.Choices{Name: "my-shop", Django: "6.1.2", Database: config.PostgresDocker, Stack: config.StackTailwindHTMX, AddOns: everyAddOnBut(config.Ninja), App: "catalog"},
		toolchain: pyenv.UV,
		node:      render.Pnpm,
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
