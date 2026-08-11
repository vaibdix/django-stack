package scaffold

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

var joinPath = filepath.Join

func (e *Env) Generate(ctx context.Context) error {
	e.ctx = ctx

	if err := e.validate(); err != nil {
		return err
	}

	projectPath, err := e.prepareProjectDirectory()
	if err != nil {
		return err
	}

	for _, s := range plan() {
		if !s.applies(e.Cfg) {
			continue
		}

		e.Progress(s.label)

		if err := s.run(e, projectPath); err != nil {

			return fmt.Errorf("%s: %w", s.name, err)
		}
	}

	e.reportPerformanceTip()
	e.Step("Django project setup complete!")
	return nil
}

func (e *Env) validate() error {
	if e.Cfg.ProjectName == "" {
		return fmt.Errorf("project name cannot be empty")
	}
	if err := e.Cfg.ValidateProjectName(); err != nil {
		return fmt.Errorf("invalid project name '%s': %w", e.Cfg.ProjectName, err)
	}
	if err := e.Cfg.ValidateAppName(); err != nil {
		return fmt.Errorf("invalid app name '%s': %w", e.Cfg.AppName, err)
	}
	return nil
}

func (e *Env) prepareProjectDirectory() (string, error) {
	projectPath := e.Cfg.ProjectName
	if !filepath.IsAbs(projectPath) {
		wd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("failed to get working directory: %w", err)
		}
		projectPath = filepath.Join(wd, e.Cfg.ProjectName)
	}

	if err := checkDirectoryIsUsable(projectPath); err != nil {
		return "", err
	}

	if err := os.MkdirAll(projectPath, 0755); err != nil {
		return "", fmt.Errorf("failed to create project directory: %w", err)
	}
	e.Step(fmt.Sprintf("Project directory created: %s", projectPath))

	return projectPath, nil
}

func checkDirectoryIsUsable(path string) error {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to inspect project directory: %w", err)
	}

	if !info.IsDir() {
		return fmt.Errorf("project path exists and is not a directory: %s", path)
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return fmt.Errorf("failed to inspect project directory: %w", err)
	}
	if len(entries) > 0 {
		return fmt.Errorf("project directory already exists and is not empty: %s", path)
	}
	return nil
}

func (e *Env) writeDevStartScript(projectPath string) error {
	scriptPath := filepath.Join(projectPath, devStartScriptName())
	script := generateStartScript(projectPath, findVSCode(), e.Cfg.SetupTailwind)

	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		e.Step("Warning: Could not create development start script: " + err.Error())
		return nil
	}

	e.Step(fmt.Sprintf("Created: %s", filepath.Base(scriptPath)))
	return nil
}
