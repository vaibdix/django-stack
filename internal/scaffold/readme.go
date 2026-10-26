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

func (p project) readmeData(uv bool) readmeData {
	data := readmeData{
		ProjectName:   p.cfg.Name,
		ModuleName:    p.cfg.ModuleName(),
		AppName:       p.cfg.App,
		PythonVersion: p.pythonToolingMetadata().MypyPythonVersion,
		DatabaseLabel: p.cfg.Database.Spec().Label,
		TestCommand:   p.testCommand(uv),
		AdminURL:      "admin/",
		Docker:        p.cfg.Database.UsesDocker(),
		AccountEmail:  p.sendsAccountEmail(),
		DjangoAuth:    p.cfg.CreateTemplates(),
		Templates:     p.cfg.CreateTemplates(),
		Node:          p.node.Name,
		UV:            uv,
		Python:        pythonCommand(uv),
		Tool:          toolCommand(uv),
		DeployPython:  deployCommand(uv, pythonCommand(uv)),
		DeployTool:    deployCommand(uv, toolCommand(uv)),
	}
	for _, c := range p.contributions() {
		if c.readme != nil {
			c.readme(&data)
		}
	}
	return data
}

func (p project) testCommand(uv bool) string {
	return strings.Join(append([]string{pythonCommand(uv), "manage.py", "test"}, p.testSettings()...), " ")
}

func (e *Env) writeReadme(ctx context.Context) error {
	if err := e.emit(e.readmeFiles()); err != nil {
		return err
	}

	e.Step("Created: README.md (setup, tests, linting and deployment)")
	return nil
}

func (p project) readmeFiles() fileSet {
	files := fileSet{}
	files.add("README.md", p.renderReadme(p.toolchain == pyenv.UV), fileMode)
	return files
}

func (p project) renderReadme(uv bool) string {
	return render("readme.md.tmpl", p.readmeData(uv))
}
