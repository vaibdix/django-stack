package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
)

type observabilityData struct {
	ModuleName  string
	ProjectName string
	Sentry      bool
	Structlog   bool
}

func (e *Env) observabilityData() observabilityData {
	return observabilityData{
		ModuleName:  e.Cfg.ModuleName(),
		ProjectName: e.Cfg.ProjectName,
		Sentry:      e.Cfg.SetupSentry,
		Structlog:   e.Cfg.SetupStructlog,
	}
}

func (e *Env) usesObservability() bool {
	return e.Cfg.SetupSentry || e.Cfg.SetupStructlog
}

func (e *Env) applyObservabilityAppConfig(projectPath string) error {
	if !e.usesObservability() {
		return nil
	}

	data := e.observabilityData()
	moduleDir := filepath.Join(projectPath, data.ModuleName)

	files := map[string]string{
		"observability.py": render("observability.py.tmpl", data),
		"apps.py":          render("project_apps.py.tmpl", data),
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(moduleDir, name), []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write %s/%s: %w", data.ModuleName, name, err)
		}
	}

	se, err := e.settingsEditor(projectPath)
	if err != nil {
		return err
	}
	appConfig := data.ModuleName + ".apps.ProjectConfig"
	if !se.Contains(appConfig) {
		if err := se.AddToList(InstalledApps, appConfig); err != nil {
			return fmt.Errorf("failed to register %s: %w", appConfig, err)
		}
	}
	if err := se.Save(); err != nil {
		return err
	}

	e.Step("Sentry and structlog start in AppConfig.ready(), not at settings import.")
	return nil
}
