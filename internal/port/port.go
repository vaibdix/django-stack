package port

import (
	"context"
	"io"
	"io/fs"
	"time"

	"github.com/vaibdix/django-stack/internal/config"
	"github.com/vaibdix/django-stack/internal/event"
)

type Runner interface {
	Run(ctx context.Context, dir, name string, args ...string) ([]byte, error)
}

type LineRunner interface {
	Runner
	RunLines(ctx context.Context, dir string, line func(string), name string, args ...string) ([]byte, error)
}

type FS interface {
	Stat(name string) (fs.FileInfo, error)
	ReadDir(name string) ([]fs.DirEntry, error)
	ReadFile(name string) ([]byte, error)
	MkdirAll(name string) error
	RemoveAll(name string) error
	WriteIn(root, rel string, data []byte, mode fs.FileMode) error
}

type Index interface {
	FileHashes(ctx context.Context, index, project, version string) ([]string, error)
}

type Docker interface {
	Ready(ctx context.Context, project string, db config.Database) error
	Start(ctx context.Context, notes event.Notes, dir, project string, db config.Database) error
	Stop(ctx context.Context, dir string) error
	FreePort(preferred int) (string, error)
}

type Ports struct {
	FS     FS
	Run    Runner
	Index  Index
	Docker Docker
	Clock  func() time.Time
	Rand   io.Reader
}
