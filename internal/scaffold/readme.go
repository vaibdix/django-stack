package scaffold

import (
	"context"
	"strings"

	"github.com/vaibdix/django-stack/internal/pyenv"
)

type readmeData struct {
	ProjectName   string
	ModuleName    string
	AppName       string
	PythonVersion string
	DatabaseLabel string
	TestCommand   string
	AdminURL      string
	Docker        bool
	Pydantic      bool
	MultiEnv      bool
	AccountEmail  bool
	REST          bool
	DRF           bool
	Ninja         bool
	DjangoAuth    bool
	AllAuth       bool
	Ruff          bool
	Mypy          bool
	PreCommit     bool
	Sentry        bool
	Templates     bool
	Tailwind      bool
	Node          string
	UV            bool
	Python        string
	Tool          string
	DeployPython  string
	DeployTool    string
}

func pythonCommand(uv bool) string {
	if uv {
		return "uv run python"
	}
	return pyenv.InterpreterRel("linux")
}

func toolCommand(uv bool) string {
	if uv {
		return "uv run"
	}
	return pyenv.InterpreterRel("linux") + " -m"
}

func deployCommand(uv bool, command string) string {
	if uv {
		return strings.Replace(command, "uv run", "uv run --no-dev", 1)
	}
	return command
}

func (e *Env) readmeData(uv bool) readmeData {
	data := readmeData{
		ProjectName:   e.cfg.Name,
		ModuleName:    e.cfg.ModuleName(),
		AppName:       e.cfg.App,
		PythonVersion: e.pythonToolingMetadata().MypyPythonVersion,
		DatabaseLabel: e.cfg.Database.Spec().Label,
		TestCommand:   e.testCommand(uv),
		AdminURL:      "admin/",
		Docker:        e.cfg.Database.UsesDocker(),
		AccountEmail:  e.sendsAccountEmail(),
		DjangoAuth:    e.cfg.CreateTemplates(),
		Templates:     e.cfg.CreateTemplates(),
		Node:          e.node.Name,
		UV:            uv,
		Python:        pythonCommand(uv),
		Tool:          toolCommand(uv),
		DeployPython:  deployCommand(uv, pythonCommand(uv)),
		DeployTool:    deployCommand(uv, toolCommand(uv)),
	}
	for _, c := range e.contributions() {
		if c.readme != nil {
			c.readme(&data)
		}
	}
	return data
}

func (e *Env) testCommand(uv bool) string {
	return strings.Join(append([]string{pythonCommand(uv), "manage.py", "test"}, e.testSettings()...), " ")
}

func (e *Env) writeReadme(ctx context.Context) error {
	content := e.renderReadme(e.toolchain == pyenv.UV)

	if err := e.write("README.md", []byte(content), fileMode); err != nil {
		return err
	}

	e.Step("Created: README.md (setup, tests, linting and deployment)")
	return nil
}

func (e *Env) renderReadme(uv bool) string {
	return render("readme.md.tmpl", e.readmeData(uv))
}
