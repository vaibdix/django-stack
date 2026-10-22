package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
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
	return readmeData{
		ProjectName:   e.cfg.ProjectName,
		ModuleName:    e.cfg.ModuleName(),
		AppName:       e.cfg.AppName,
		PythonVersion: e.pythonToolingMetadata().MypyPythonVersion,
		DatabaseLabel: e.cfg.DatabaseType.Spec().Label,
		TestCommand:   e.testCommand(uv),
		AdminURL:      "admin/",
		Docker:        e.cfg.DatabaseType.UsesDocker(),
		Pydantic:      e.cfg.SetupPydanticConf,
		MultiEnv:      e.cfg.SetupMultiEnv,
		AccountEmail:  e.sendsAccountEmail(),
		REST:          e.cfg.HasAPI(),
		DRF:           e.cfg.SetupRestFramework,
		Ninja:         e.cfg.SetupNinja,
		DjangoAuth:    e.cfg.CreateTemplates && !e.cfg.SetupAllAuth,
		AllAuth:       e.cfg.SetupAllAuth,
		Ruff:          e.cfg.SetupRuff,
		Mypy:          e.cfg.SetupMypy,
		PreCommit:     e.cfg.SetupPreCommit,
		Sentry:        e.cfg.SetupSentry,
		Templates:     e.cfg.CreateTemplates,
		Tailwind:      e.cfg.SetupTailwind,
		Node:          e.node.Name,
		UV:            uv,
		Python:        pythonCommand(uv),
		Tool:          toolCommand(uv),
		DeployPython:  deployCommand(uv, pythonCommand(uv)),
		DeployTool:    deployCommand(uv, toolCommand(uv)),
	}
}

func (e *Env) testCommand(uv bool) string {
	command := pythonCommand(uv) + " manage.py test"
	if e.cfg.SetupMultiEnv {
		return command + " --settings=" + e.cfg.ModuleName() + ".settings.test"
	}
	return command
}

func (e *Env) writeReadme(projectPath string) error {
	content := e.renderReadme(e.toolchain == pyenv.UV)

	path := filepath.Join(projectPath, "README.md")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write README.md: %w", err)
	}

	e.Step("Created: README.md (setup, tests, linting and deployment)")
	return nil
}

func (e *Env) renderReadme(uv bool) string {
	return render("readme.md.tmpl", e.readmeData(uv))
}
