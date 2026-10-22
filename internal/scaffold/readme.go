package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vaibdix/django-cli/internal/config"
	"github.com/vaibdix/django-cli/internal/scaffold/pyenv"
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
	return ".venv/bin/python"
}

func toolCommand(uv bool) string {
	if uv {
		return "uv run"
	}
	return ".venv/bin/python -m"
}

func deployCommand(uv bool, command string) string {
	if uv {
		return strings.Replace(command, "uv run", "uv run --no-dev", 1)
	}
	return command
}

func (e *Env) readmeData(uv bool) readmeData {
	return readmeData{
		ProjectName:   e.Cfg.ProjectName,
		ModuleName:    e.Cfg.ModuleName(),
		AppName:       e.Cfg.AppName,
		PythonVersion: e.pythonToolingMetadata().MypyPythonVersion,
		DatabaseLabel: databaseLabel(e.Cfg.DatabaseType),
		TestCommand:   e.testCommand(uv),
		AdminURL:      "admin/",
		Docker:        e.Cfg.DatabaseType.UsesDocker(),
		Pydantic:      e.Cfg.SetupPydanticConf,
		MultiEnv:      e.Cfg.SetupMultiEnv,
		AccountEmail:  e.sendsAccountEmail(),
		REST:          e.Cfg.HasAPI(),
		DRF:           e.Cfg.SetupRestFramework,
		Ninja:         e.Cfg.SetupNinja,
		DjangoAuth:    e.Cfg.CreateTemplates && !e.Cfg.SetupAllAuth,
		AllAuth:       e.Cfg.SetupAllAuth,
		Ruff:          e.Cfg.SetupRuff,
		Mypy:          e.Cfg.SetupMypy,
		PreCommit:     e.Cfg.SetupPreCommit,
		Templates:     e.Cfg.CreateTemplates,
		Tailwind:      e.Cfg.SetupTailwind,
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
	if e.Cfg.SetupMultiEnv {
		return command + " --settings=" + e.Cfg.ModuleName() + ".settings.test"
	}
	return command
}

func databaseLabel(db config.Database) string {
	switch db {
	case config.PostgresDocker:
		return "PostgreSQL"
	case config.MySQLDocker:
		return "MySQL"
	default:
		return "SQLite"
	}
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
