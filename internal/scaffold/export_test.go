package scaffold

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/vaibdix/django-stack/internal/config"
	"github.com/vaibdix/django-stack/internal/livetest"
)

func injectEnvConfigInto(_ string, e *Env) string {
	return e.settingsText()
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
	set.addAll(filepath.Join(e.cfg.App().String(), "api"), apiFiles)
	return e.emit(set)
}

func (e *Env) at(dir string) *Env {
	e.dir = dir
	return e
}

func writeVSCodeConfig(root string, cfg config.Choices, manager NodeManager) error {
	return (&Env{dir: root, cfg: mustSpec(cfg), node: manager}).writeVSCodeConfig()
}

func (w workspace) write(root string) error {
	files, err := w.files()
	if err != nil {
		return err
	}
	return (&Env{dir: root}).emit(files)
}

func (e *Env) writeConfModule() error {
	return e.emit(e.confFiles())
}

func (e *Env) configureSettingsAt(path string) error {
	return os.WriteFile(path, []byte(e.settingsText()), 0o644)
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
		livetest.MissingTool(t, "uv is not installed")
	}
	return uv
}

func (e *Env) writePyprojectWith(addOns ...config.AddOn) error {
	e.cfg = withAddOns(e.cfg, addOns...)
	return e.write(pyprojectFile, []byte(e.renderPyproject(nil, nil)), fileMode)
}

func basePyFor(t *testing.T, e *Env, _ string) string {
	t.Helper()
	split := &project{cfg: e.cfg, resolved: e.resolved}
	split.cfg = withAddOns(split.cfg, config.MultiEnv)
	return split.settingsText()
}

func vscodeExtensionsJSON(cfg config.Choices) string {
	data, err := marshalVSCode(extensionsFile(mustSpec(cfg)))
	if err != nil {
		panic(err)
	}
	return string(data)
}

func generateBookModel(appName string, htmlViews bool) string {
	return render("models.py.tmpl", bookModelData{AppName: appName, HTMLViews: htmlViews})
}

func (e *Env) writeBookApp() error {
	if err := e.emit(e.bookFiles()); err != nil {
		return err
	}
	e.Step("Created the book models, admin, sample data command and pages.")
	return nil
}

func generateHtmxSupport() string { return htmxSupport(false) }

func generateAPIPackage(appName string) map[string]string {
	return apiPackage(apiPackageData{AppName: appName})
}

func (p project) settingsText() string {
	text, err := p.renderSettingsModule()
	if err != nil {
		panic(err)
	}
	return text
}

func (p project) urlsText() string {
	text, err := p.buildProjectURLsContent()
	if err != nil {
		panic(err)
	}
	return text
}
