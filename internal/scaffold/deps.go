package scaffold

import (
	"context"
	"crypto/rand"
	"io"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/vaibdix/django-stack/internal/command"
	"github.com/vaibdix/django-stack/internal/pyenv"
	"github.com/vaibdix/django-stack/internal/pypi"
	"github.com/vaibdix/django-stack/internal/releases"
)

type deps struct {
	runModule     func(ctx context.Context, e *Env, dir, python, module string, args ...string) ([]byte, error)
	scratchTools  func(ctx context.Context, e *Env, packages []string) (string, func() error, error)
	hashes        func(ctx context.Context, index, project, version string) ([]string, error)
	lookPath      func(name string) (string, error)
	stat          func(name string) (fs.FileInfo, error)
	startDetached func(ctx context.Context, dir, name string, args ...string) error
	nodeManager   func(ctx context.Context, run command.Runner) (NodeManager, bool)
	entropy       io.Reader
}

func (d deps) withDefaults() deps {
	if d.runModule == nil {
		d.runModule = runPythonModule
	}
	if d.scratchTools == nil {
		d.scratchTools = installScratchFormatters
	}
	if d.hashes == nil {
		d.hashes = indexHashes
	}
	if d.lookPath == nil {
		d.lookPath = exec.LookPath
	}
	if d.stat == nil {
		d.stat = os.Stat
	}
	if d.startDetached == nil {
		d.startDetached = startDetached
	}
	if d.nodeManager == nil {
		d.nodeManager = d.detectNodeManager
	}
	if d.entropy == nil {
		d.entropy = rand.Reader
	}
	return d
}

func (d deps) host() pyenv.Host {
	return pyenv.Host{GOOS: runtime.GOOS, LookPath: d.lookPath, Stat: d.stat, Getenv: os.Getenv}
}

func (e *Env) dependencies() deps {
	return e.deps.withDefaults()
}

func indexHashes(ctx context.Context, index, project, version string) ([]string, error) {
	return pypi.FileHashes(ctx, &http.Client{Timeout: 30 * time.Second}, index, project, version)
}

func (e *Env) indexFor() string {
	return releases.IndexFor(e.toolchain, os.Getenv)
}
