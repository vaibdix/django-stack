package adapter

import (
	"context"
	"crypto/rand"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/vaibdix/django-stack/internal/command"
	"github.com/vaibdix/django-stack/internal/config"
	"github.com/vaibdix/django-stack/internal/dbdocker"
	"github.com/vaibdix/django-stack/internal/event"
	"github.com/vaibdix/django-stack/internal/port"
	"github.com/vaibdix/django-stack/internal/pypi"
)

const dirMode fs.FileMode = 0o755

const indexTimeout = 30 * time.Second

type OSFS struct{}

func (OSFS) Stat(name string) (fs.FileInfo, error) { return os.Stat(name) }

func (OSFS) ReadDir(name string) ([]fs.DirEntry, error) { return os.ReadDir(name) }

func (OSFS) ReadFile(name string) ([]byte, error) { return os.ReadFile(name) }

func (OSFS) MkdirAll(name string) error { return os.MkdirAll(name, dirMode) }

func (OSFS) RemoveAll(name string) error { return os.RemoveAll(name) }

func (OSFS) Rename(from, to string) error { return os.Rename(from, to) }

func (OSFS) WriteIn(root, rel string, data []byte, mode fs.FileMode) error {
	dir, err := os.OpenRoot(root)
	if err != nil {
		return fmt.Errorf("open the project directory: %w", err)
	}
	defer func() { _ = dir.Close() }()

	if parent := filepath.Dir(rel); parent != "." {
		if err := dir.MkdirAll(parent, dirMode); err != nil {
			return fmt.Errorf("create %s: %w", filepath.ToSlash(parent), err)
		}
	}
	if err := dir.WriteFile(rel, data, mode); err != nil {
		return fmt.Errorf("write %s: %w", filepath.ToSlash(rel), err)
	}
	return nil
}

type PyPI struct{ Client *http.Client }

func (p PyPI) FileHashes(ctx context.Context, index, project, version string) ([]string, error) {
	client := p.Client
	if client == nil {
		client = &http.Client{Timeout: indexTimeout}
	}
	return pypi.FileHashes(ctx, client, index, project, version)
}

type DockerCLI struct{ Run port.Runner }

func (d DockerCLI) Ready(ctx context.Context, project string, db config.Database) error {
	return dbdocker.Ready(ctx, d.Run, project, db)
}

func (d DockerCLI) Start(ctx context.Context, notes event.Notes, dir, project string, db config.Database) error {
	return dbdocker.Start(ctx, d.Run, notes, dir, project, db)
}

func (d DockerCLI) Stop(ctx context.Context, dir string) error {
	return dbdocker.Stop(ctx, d.Run, dir)
}

func (DockerCLI) FreePort(preferred int) (string, error) { return dbdocker.AvailablePort(preferred) }

func Complete(p port.Ports) port.Ports {
	if p.Run == nil {
		p.Run = command.NewOS()
	}
	if p.FS == nil {
		p.FS = OSFS{}
	}
	if p.Index == nil {
		p.Index = PyPI{}
	}
	if p.Docker == nil {
		p.Docker = DockerCLI{Run: p.Run}
	}
	if p.Clock == nil {
		p.Clock = time.Now
	}
	if p.Rand == nil {
		p.Rand = rand.Reader
	}
	return p
}
