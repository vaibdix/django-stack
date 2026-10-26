package scaffold

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/vaibdix/django-stack/internal/config"
)

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
	e.dir = projectPath
	if err := e.writeBookApp(); err != nil {
		return err
	}
	set := fileSet{}
	set.addAll(filepath.Join(e.cfg.App, "api"), apiFiles)
	return e.emit(set)
}

func (e *Env) at(dir string) *Env {
	e.dir = dir
	return e
}

func writeVSCodeConfig(root string, cfg config.Choices, manager NodeManager) error {
	return (&Env{dir: root, cfg: cfg, node: manager}).writeVSCodeConfig()
}

func (w workspace) write(root string) error {
	return (&Env{dir: root}).writeWorkspace(w)
}

func newSettingsEditor(path string) (*settingsEditor, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return &settingsEditor{content: string(data), save: func(content string) error {
		return os.WriteFile(path, []byte(content), 0o644)
	}}, nil
}

func (e *Env) configureSettingsAt(path string) error {
	se, err := newSettingsEditor(path)
	if err != nil {
		return err
	}
	return e.configureSettingsIn(se)
}

func newPyprojectEditor(projectPath, projectName string, metadata pythonToolingMetadata) (*pyprojectEditor, error) {
	return (&Env{dir: projectPath, cfg: config.Choices{Name: projectName}}).pyproject(metadata)
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
	if e.resolved.SecretKey == "" {
		key, err := generateSecretKey(e.randomness())
		if err != nil {
			t.Fatal(err)
		}
		e.resolved.SecretKey = key
	}
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
