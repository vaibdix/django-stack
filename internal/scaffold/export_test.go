package scaffold

import (
	"context"
	"testing"

	"github.com/vaibdix/django-cli/internal/config"
)

func withPythonModuleRunner(hook func(ctx context.Context, e *Env, projectPath, module string, args ...string) ([]byte, error)) func() {
	original := runPythonModuleCommand
	runPythonModuleCommand = hook
	return func() { runPythonModuleCommand = original }
}

func addProjectContextProcessorTo(settingsContent, projectModuleName string) (string, error) {
	se := &SettingsEditor{content: settingsContent}
	if err := se.AddProjectContextProcessor(projectModuleName); err != nil {
		return "", err
	}
	return se.Content(), nil
}

func configureTemplatesDirsIn(settingsContent string) (string, error) {
	se := &SettingsEditor{content: settingsContent}
	if err := se.ConfigureTemplatesDirs(); err != nil {
		return "", err
	}
	return se.Content(), nil
}

func injectEnvConfigInto(settingsContent string, e *Env) string {
	se := &SettingsEditor{content: settingsContent}
	if err := se.InjectEnvConfig(e.envConfig()); err != nil {
		panic(err)
	}
	return se.Content()
}

func pythonToolingMetadataOf(e *Env) (requiresPython, ruffTarget, mypyVersion string) {
	m := e.pythonToolingMetadata()
	return m.RequiresPython, m.RuffTargetVersion, m.MypyPythonVersion
}

func (e *Env) writeExampleApp(projectPath string, apiFiles map[string]string) error {
	if err := e.writeBookApp(projectPath); err != nil {
		return err
	}
	return e.writeAPIPackage(projectPath, apiFiles)
}

func basePyFor(t *testing.T, e *Env, settings string) string {
	t.Helper()
	base, err := e.generateBasePy(settings)
	if err != nil {
		t.Fatalf("generateBasePy: %v", err)
	}
	return base
}

// withCredentials fills in what prepareDatabaseCredentials chooses before the
// .env file is written, with fixed values so goldens stay stable.
func withCredentials(cfg config.Project) config.Project {
	switch cfg.DatabaseType {
	case config.PostgresDocker:
		cfg.PostgresDockerPassword = orDefault(cfg.PostgresDockerPassword, "pgapppassword001")
		cfg.PostgresDockerRootPassword = orDefault(cfg.PostgresDockerRootPassword, "pgrootpassword01")
		cfg.PostgresDockerPort = orDefault(cfg.PostgresDockerPort, "5432")
	case config.MySQLDocker:
		cfg.MySQLDockerPassword = orDefault(cfg.MySQLDockerPassword, "myapppassword001")
		cfg.MySQLDockerRootPassword = orDefault(cfg.MySQLDockerRootPassword, "myrootpassword01")
		cfg.MySQLDockerPort = orDefault(cfg.MySQLDockerPort, "3306")
	}
	return cfg
}

func orDefault(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func envFileFor(t *testing.T, e *Env) string {
	t.Helper()
	content, err := e.renderEnvFileContent()
	if err != nil {
		t.Fatalf("renderEnvFileContent: %v", err)
	}
	return content
}
