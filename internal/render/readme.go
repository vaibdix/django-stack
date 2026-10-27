package render

import (
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

func (p Project) readmeData(uv bool) readmeData {
	data := readmeData{
		ProjectName:   p.Cfg.Name().String(),
		ModuleName:    p.Cfg.ModuleName(),
		AppName:       p.Cfg.App().String(),
		PythonVersion: p.PythonToolingMetadata().MypyPythonVersion,
		DatabaseLabel: p.Cfg.Database().Spec().Label,
		TestCommand:   p.testCommand(uv),
		AdminURL:      "admin/",
		Docker:        p.Cfg.Database().UsesDocker(),
		AccountEmail:  p.sendsAccountEmail(),
		DjangoAuth:    p.Cfg.CreateTemplates(),
		Templates:     p.Cfg.CreateTemplates(),
		Node:          p.Node.Name,
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

func (p Project) testCommand(uv bool) string {
	return strings.Join(append([]string{pythonCommand(uv), "manage.py", "test"}, p.testSettings()...), " ")
}

func (p Project) readmeFiles() FileSet {
	files := FileSet{}
	files.add("README.md", p.renderReadme(p.Toolchain == pyenv.UV), FileMode)
	return files
}

func (p Project) renderReadme(uv bool) string {
	return Render("readme.md.tmpl", p.readmeData(uv))
}
