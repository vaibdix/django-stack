package scaffold

import (
	"github.com/vaibdix/django-stack/internal/config"
)

func (p project) structlogSettingsBlock() string {
	return render("structlog.py.tmpl", struct{ Debug, Level, Module string }{"DEBUG", logLevelExpression, p.cfg.ModuleName()})
}

const logLevelExpression = `"DEBUG" if DEBUG else "INFO"`

func structlogFeature() feature {
	return feature{
		enabled: has(config.Structlog),
		contribute: func(p project) contribution {
			return contribution{
				runtime: []string{pkgStructlog, pkgDjangoStructlog},
				settings: settingsContribution{
					imports:    []string{"import structlog"},
					apps:       []entry{{name: appStructlog, after: lastDjangoApp, rank: rankStructlog}},
					middleware: []entry{{name: middlewareStructlog, after: lastDjangoMiddleware, rank: rankStructlog}},
					blocks:     []block{{rank: rankStructlog, text: p.structlogSettingsBlock()}},
				},
			}
		},
	}
}
