package scaffold

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/vaibdix/django-stack/internal/config"
)

func injectEnvConfigInto(_ string, e *Env) string {
	return e.renderSettingsModule()
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

func (e *Env) configureSettingsAt(path string) error {
	return os.WriteFile(path, []byte(e.renderSettingsModule()), 0o644)
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

func (e *Env) writePyprojectWith(addOns ...config.AddOn) error {
	for _, addOn := range addOns {
		e.cfg.AddOns = e.cfg.AddOns.With(addOn)
	}
	return e.write(pyprojectFile, []byte(e.renderPyproject(nil, nil)), fileMode)
}

func basePyFor(t *testing.T, e *Env, _ string) string {
	t.Helper()
	split := &Env{cfg: e.cfg, resolved: e.resolved}
	split.cfg.AddOns = split.cfg.AddOns.With(config.MultiEnv)
	return split.renderSettingsModule()
}

func vscodeExtensionsJSON(cfg config.Choices) string {
	data, err := marshalVSCode(extensionsFile(cfg))
	if err != nil {
		panic(err)
	}
	return string(data)
}

func generateBookModel(appName string, htmlViews bool) string {
	return render("models.py.tmpl", bookModelData{AppName: appName, HTMLViews: htmlViews})
}
