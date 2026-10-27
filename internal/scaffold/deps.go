package scaffold

import (
	"context"
	"io/fs"
	"os"
	"os/exec"
	"runtime"

	"github.com/vaibdix/django-stack/internal/command"
	"github.com/vaibdix/django-stack/internal/pyenv"
	"github.com/vaibdix/django-stack/internal/releases"
	"github.com/vaibdix/django-stack/internal/render"
)

type deps struct {
	lookPath      func(name string) (string, error)
	stat          func(name string) (fs.FileInfo, error)
	startDetached func(ctx context.Context, dir, name string, args ...string) error
	nodeManager   func(ctx context.Context, run command.Runner) (render.NodeManager, bool)
}

func (d deps) withDefaults() deps {
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
	return d
}

func (d deps) host() pyenv.Host {
	return pyenv.Host{GOOS: runtime.GOOS, LookPath: d.lookPath, Stat: d.stat, Getenv: os.Getenv}
}

func (e *Env) dependencies() deps {
	return e.deps.withDefaults()
}

func (e *Env) indexFor() string {
	return releases.IndexFor(e.Toolchain, os.Getenv)
}
