package render

import (
	"github.com/vaibdix/django-stack/internal/config"
)

func htmxFeature() feature {
	return feature{
		enabled: config.Spec.UsesHTMX,
		contribute: func(Project) contribution {
			return contribution{
				runtime: []string{PkgHTMX},
				settings: settingsContribution{
					apps:       []entry{{name: appHTMX, after: lastDjangoApp, rank: rankHTMX}},
					middleware: []entry{{name: middlewareHTMX, after: lastDjangoMiddleware, rank: rankHTMX}},
				},
			}
		},
	}
}
