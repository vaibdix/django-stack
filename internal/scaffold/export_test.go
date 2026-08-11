package scaffold

import "context"

func withFormattingInstallHook(hook func(e *Env, projectPath string, packages ...string) error) func() {
	original := installFormattingPackages
	installFormattingPackages = hook
	return func() { installFormattingPackages = original }
}

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
	se.InjectEnvConfig(e.Cfg.DatabaseType)
	return se.Content()
}

func pythonToolingMetadataOf(e *Env) (requiresPython, ruffTarget, mypyVersion string) {
	m := e.pythonToolingMetadata()
	return m.RequiresPython, m.RuffTargetVersion, m.MypyPythonVersion
}
