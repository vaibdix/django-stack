package scaffold

import (
	"context"
	"testing"
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
