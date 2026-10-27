package scaffold

import (
	"crypto/rand"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/vaibdix/django-stack/internal/config"
	"github.com/vaibdix/django-stack/internal/livetest"
)

func writeTo(root string, files fileSet) error {
	for rel, file := range files {
		path := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(path, file.Content, file.Mode); err != nil {
			return err
		}
	}
	return nil
}

func injectEnvConfigInto(_ string, p *project) string {
	return p.settingsText()
}

func pythonToolingMetadataOf(p *project) (requiresPython, ruffTarget, mypyVersion string) {
	m := p.pythonToolingMetadata()
	return m.RequiresPython, m.RuffTargetVersion, m.MypyPythonVersion
}

func (p *project) writeExampleApp(projectPath string, apiFiles map[string]string) error {
	set := p.bookFiles()
	set.addAll(filepath.Join(p.cfg.App().String(), "api"), apiFiles)
	return writeTo(projectPath, set)
}

func (e *Env) at(dir string) *Env {
	e.dir = dir
	return e
}

func writeVSCodeConfig(root string, cfg config.Choices, manager NodeManager) error {
	return project{cfg: mustSpec(cfg), node: manager}.workspace().write(root)
}

func (w workspace) write(root string) error {
	files, err := w.files()
	if err != nil {
		return err
	}
	return writeTo(root, files)
}

func (p *project) writeConfModule(root string) error {
	return writeTo(root, p.confFiles())
}

func (p *project) configureSettingsAt(path string) error {
	return os.WriteFile(path, []byte(p.settingsText()), 0o644)
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

func envFileFor(t *testing.T, p *project) string {
	t.Helper()
	return envFileWith(t, p, rand.Reader)
}

func envFileWith(t *testing.T, p *project, random io.Reader) string {
	t.Helper()
	if p.resolved.SecretKey == "" {
		key, err := generateSecretKey(random)
		if err != nil {
			t.Fatal(err)
		}
		p.resolved.SecretKey = key
	}
	content, err := p.renderEnvFileContent()
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

func (p *project) writePyprojectWith(root string, addOns ...config.AddOn) error {
	p.cfg = withAddOns(p.cfg, addOns...)
	return os.WriteFile(filepath.Join(root, pyprojectFile), []byte(p.renderPyproject(nil, nil)), fileMode)
}

func basePyFor(t *testing.T, p *project, _ string) string {
	t.Helper()
	split := &project{cfg: p.cfg, resolved: p.resolved}
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

func (p *project) writeBookApp(root string) error {
	return writeTo(root, p.bookFiles())
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
