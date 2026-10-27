package scaffold

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/vaibdix/django-stack/internal/config"
	"github.com/vaibdix/django-stack/internal/livetest"
	"github.com/vaibdix/django-stack/internal/render"
)

func writeTo(root string, files render.FileSet) error {
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

func (e *Env) at(dir string) *Env {
	e.dir = dir
	return e
}

func writeVSCodeConfig(root string, cfg config.Choices, manager render.NodeManager) error {
	return writeWorkspace(render.Project{Cfg: mustSpec(cfg), Node: manager}.Workspace(), root)
}

func writeWorkspace(w render.Workspace, root string) error {
	files, err := w.Files()
	if err != nil {
		return err
	}
	return writeTo(root, files)
}

func writeConfModule(p *render.Project, root string) error {
	return writeTo(root, p.ConfFiles())
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

func writePyprojectWith(p *render.Project, root string, addOns ...config.AddOn) error {
	p.Cfg = withAddOns(p.Cfg, addOns...)
	return os.WriteFile(filepath.Join(root, render.PyprojectFile), []byte(p.RenderPyproject(nil, nil)), render.FileMode)
}

func basePyFor(t *testing.T, p *render.Project, _ string) string {
	t.Helper()
	split := &render.Project{Cfg: p.Cfg, Resolved: p.Resolved}
	split.Cfg = withAddOns(split.Cfg, config.MultiEnv)
	return settingsText(split)
}

func settingsText(p *render.Project) string {
	text, err := p.RenderSettingsModule()
	if err != nil {
		panic(err)
	}
	return text
}
