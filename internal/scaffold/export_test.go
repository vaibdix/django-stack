package scaffold

import (
	"context"
	"os/exec"
	"testing"

	"github.com/vaibdix/django-stack/internal/config"
)

func withPythonModuleRunner(hook func(ctx context.Context, e *Env, projectPath, python, module string, args ...string) ([]byte, error)) func() {
	original := runPythonModuleCommand
	runPythonModuleCommand = hook
	return func() { runPythonModuleCommand = original }
}

func withScratchFormatters(hook func(ctx context.Context, e *Env, packages []string) (string, func() error, error)) func() {
	original := installScratchFormatters
	installScratchFormatters = hook
	return func() { installScratchFormatters = original }
}

func addProjectContextProcessorTo(settingsContent, projectModuleName string) (string, error) {
	se := &settingsEditor{content: settingsContent}
	if err := se.AddProjectContextProcessor(projectModuleName); err != nil {
		return "", err
	}
	return se.Content(), nil
}

func configureTemplatesDirsIn(settingsContent string) (string, error) {
	se := &settingsEditor{content: settingsContent}
	if err := se.ConfigureTemplatesDirs(); err != nil {
		return "", err
	}
	return se.Content(), nil
}

func injectEnvConfigInto(settingsContent string, e *Env) string {
	se := &settingsEditor{content: settingsContent}
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

var testCredentials = map[config.Database]struct{ password, rootPassword, port string }{
	config.PostgresDocker: {"pgapppassword001", "pgrootpassword01", "5432"},
	config.MySQLDocker:    {"myapppassword001", "myrootpassword01", "3306"},
}

func credentialsFor(db config.Database, preset config.DatabaseCredentials) config.Resolved {
	if c, ok := testCredentials[db]; ok {
		preset.Password = orDefault(preset.Password, c.password)
		preset.RootPassword = orDefault(preset.RootPassword, c.rootPassword)
		preset.Port = orDefault(preset.Port, c.port)
	}
	return config.Resolved{Database: preset}
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

func requireUV(t *testing.T) string {
	t.Helper()
	if testing.Short() {
		t.Skip("downloads packages through uv; run without -short")
	}
	uv, err := exec.LookPath("uv")
	if err != nil {
		t.Skip("uv is not installed")
	}
	return uv
}
