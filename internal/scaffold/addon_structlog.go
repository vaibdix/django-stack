package scaffold

import (
	"github.com/vaibdix/django-stack/internal/config"
)

func (p project) structlogSettingsBlock() string {
	return render("structlog.py.tmpl", struct{ Debug, Level, Module string }{"DEBUG", logLevelExpression, p.cfg.ModuleName()})
}

const logLevelExpression = `"DEBUG" if DEBUG else "INFO"`

func structlogAddOn() addOn {
	return addOn{
		enabled: has(config.Structlog),
		contribute: func(p project) contribution {
			return contribution{
				runtime: []string{pkgStructlog, pkgDjangoStructlog},
				settings: settingsContribution{
					imports:    []string{"import structlog"},
					apps:       []string{appStructlog},
					middleware: []string{middlewareStructlog},
					blocks:     []string{p.structlogSettingsBlock()},
				},
			}
		},
	}
}
