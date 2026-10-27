package render

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

func writeTo(root string, files FileSet) error {
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

func injectEnvConfigInto(_ string, p *Project) string {
	return settingsText(p)
}

func pythonToolingMetadataOf(p *Project) (requiresPython, ruffTarget, mypyVersion string) {
	m := p.PythonToolingMetadata()
	return m.RequiresPython, m.RuffTargetVersion, m.MypyPythonVersion
}

func writeExampleApp(p *Project, projectPath string, apiFiles map[string]string) error {
	set := p.bookFiles()
	set.addAll(filepath.Join(p.Cfg.App().String(), "api"), apiFiles)
	return writeTo(projectPath, set)
}

func writeVSCodeConfig(root string, cfg config.Choices, manager NodeManager) error {
	return writeWorkspace(Project{Cfg: mustSpec(cfg), Node: manager}.Workspace(), root)
}

func writeWorkspace(w Workspace, root string) error {
	files, err := w.Files()
	if err != nil {
		return err
	}
	return writeTo(root, files)
}

func writeConfModule(p *Project, root string) error {
	return writeTo(root, p.ConfFiles())
}

func configureSettingsAt(p *Project, path string) error {
	return os.WriteFile(path, []byte(settingsText(p)), 0o644)
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

func envFileFor(t *testing.T, p *Project) string {
	t.Helper()
	return envFileWith(t, p, rand.Reader)
}

func envFileWith(t *testing.T, p *Project, random io.Reader) string {
	t.Helper()
	if p.Resolved.SecretKey == "" {
		key, err := GenerateSecretKey(random)
		if err != nil {
			t.Fatal(err)
		}
		p.Resolved.SecretKey = key
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

func writePyprojectWith(p *Project, root string, addOns ...config.AddOn) error {
	p.Cfg = withAddOns(p.Cfg, addOns...)
	return os.WriteFile(filepath.Join(root, PyprojectFile), []byte(p.RenderPyproject(nil, nil)), FileMode)
}

func basePyFor(t *testing.T, p *Project, _ string) string {
	t.Helper()
	split := &Project{Cfg: p.Cfg, Resolved: p.Resolved}
	split.Cfg = withAddOns(split.Cfg, config.MultiEnv)
	return settingsText(split)
}

func vscodeExtensionsJSON(cfg config.Choices) string {
	data, err := marshalVSCode(extensionsFile(mustSpec(cfg)))
	if err != nil {
		panic(err)
	}
	return string(data)
}

func generateBookModel(appName string, htmlViews bool) string {
	return Render("models.py.tmpl", bookModelData{AppName: appName, HTMLViews: htmlViews})
}

func writeBookApp(p *Project, root string) error {
	return writeTo(root, p.bookFiles())
}

func generateHtmxSupport() string { return htmxSupport(false) }

func generateAPIPackage(appName string) map[string]string {
	return apiPackage(apiPackageData{AppName: appName})
}

func settingsText(p *Project) string {
	text, err := p.RenderSettingsModule()
	if err != nil {
		panic(err)
	}
	return text
}

func urlsText(p *Project) string {
	text, err := p.buildProjectURLsContent()
	if err != nil {
		panic(err)
	}
	return text
}
