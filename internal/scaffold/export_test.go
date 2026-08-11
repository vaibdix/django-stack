package scaffold

import "context"

// Test helpers for this package.
//
// This file used to be test_exports.go — a normal source file, with no build
// tag, shipping 194 lines and 29 exported ...ForTest shims inside every
// released binary. They existed only because the tests lived in a separate
// directory and so could not reach unexported identifiers.
//
// The tests are in-package now, so almost all of those shims were pure
// indirection and have been inlined at their call sites. What remains is the
// handful that do real work: swapping a package-level seam and restoring it, or
// building a SettingsEditor around a literal string. Being in an _test.go file,
// none of it is compiled into the binary.

// withFormattingInstallHook replaces the package installer used by the
// formatting pass and returns a function that restores it.
//
// Not safe for parallel tests: it mutates package-level state.
func withFormattingInstallHook(hook func(e *Env, projectPath string, packages ...string) error) func() {
	original := installFormattingPackages
	installFormattingPackages = hook
	return func() { installFormattingPackages = original }
}

// withPythonModuleRunner replaces the `python -m <module>` runner so formatting
// tests do not need ruff or djlint installed.
//
// Not safe for parallel tests: it mutates package-level state.
func withPythonModuleRunner(hook func(ctx context.Context, e *Env, projectPath, module string, args ...string) ([]byte, error)) func() {
	original := runPythonModuleCommand
	runPythonModuleCommand = hook
	return func() { runPythonModuleCommand = original }
}

// The three helpers below apply one SettingsEditor edit to an in-memory
// settings.py, so the idempotency tests never touch disk.

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

// pythonToolingMetadataOf flattens the metadata struct into the three strings
// the table test asserts on.
func pythonToolingMetadataOf(e *Env) (requiresPython, ruffTarget, mypyVersion string) {
	m := e.pythonToolingMetadata()
	return m.RequiresPython, m.RuffTargetVersion, m.MypyPythonVersion
}
