package render

import (
	"github.com/vaibdix/django-stack/internal/config"
)

func (p Project) structlogSettingsBlock() string {
	return Render("structlog.py.tmpl", struct{ Debug, Level, Module string }{"DEBUG", logLevelExpression, p.Cfg.ModuleName()})
}

const logLevelExpression = `"DEBUG" if DEBUG else "INFO"`

func structlogFeature() feature {
	return feature{
		enabled: Has(config.Structlog),
		contribute: func(p Project) contribution {
			return contribution{
				runtime: []string{PkgStructlog, pkgDjangoStructlog},
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
