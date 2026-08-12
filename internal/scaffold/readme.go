package scaffold

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/vaibdix/django-cli/internal/config"
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
}

func (e *Env) readmeData() readmeData {
	return readmeData{
		ProjectName:   e.Cfg.ProjectName,
		ModuleName:    e.Cfg.ModuleName(),
		AppName:       e.Cfg.AppName,
		PythonVersion: e.pythonToolingMetadata().MypyPythonVersion,
		DatabaseLabel: databaseLabel(e.Cfg.DatabaseType),
		TestCommand:   e.testCommand(),
		AdminURL:      "admin/",
		Docker:        e.Cfg.DatabaseType.UsesDocker(),
		Pydantic:      e.Cfg.SetupPydanticConf,
		MultiEnv:      e.Cfg.SetupMultiEnv,
		REST:          e.Cfg.SetupRestFramework,
		AllAuth:       e.Cfg.SetupAllAuth,
		Ruff:          e.Cfg.SetupRuff,
		Mypy:          e.Cfg.SetupMypy,
		PreCommit:     e.Cfg.SetupPreCommit,
	}
}

func (e *Env) testCommand() string {
	if e.Cfg.SetupMultiEnv {
		return "python manage.py test --settings=" + e.Cfg.ModuleName() + ".settings.test"
	}
	return "python manage.py test"
}

func databaseLabel(db config.Database) string {
	switch db {
	case config.PostgresDocker, config.PostgresString:
		return "PostgreSQL"
	case config.MySQLDocker:
		return "MySQL"
	default:
		return "SQLite"
	}
}

func (e *Env) writeReadme(projectPath string) error {
	content := render("readme.md.tmpl", e.readmeData())

	path := filepath.Join(projectPath, "README.md")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write README.md: %w", err)
	}

	e.Step("Created: README.md (setup, tests, linting and deployment)")
	return nil
}
