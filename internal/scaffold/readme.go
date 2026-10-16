package scaffold

import (
	"fmt"
	"os"
	"path/filepath"

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
	REST          bool
	AllAuth       bool
	Ruff          bool
	Mypy          bool
	PreCommit     bool
	Templates     bool
	UV            bool
	Python        string
	Tool          string
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
		REST:          e.Cfg.HasAPI(),
		AllAuth:       e.Cfg.SetupAllAuth,
		Ruff:          e.Cfg.SetupRuff,
		Mypy:          e.Cfg.SetupMypy,
		PreCommit:     e.Cfg.SetupPreCommit,
		Templates:     e.Cfg.CreateTemplates,
		UV:            uv,
		Python:        pythonCommand(uv),
		Tool:          toolCommand(uv),
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
	content := e.renderReadme(pyenv.UVAvailable())

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
