package render

import (
	"github.com/vaibdix/django-stack/internal/config"
	"github.com/vaibdix/django-stack/internal/pyenv"
)

type Project struct {
	Cfg       config.Spec
	Resolved  config.Resolved
	Toolchain pyenv.Toolchain
	Node      NodeManager
}
