package render

import (
	"github.com/vaibdix/django-stack/internal/config"
)

func cspFeature() feature {
	return feature{
		enabled: func(c config.Spec) bool { return !c.Django().SupportsNativeCSP() },
		contribute: func(Project) contribution {
			return contribution{
				runtime: []string{pkgDjangoCSP},
				settings: settingsContribution{
					apps:       []entry{{name: appDjangoCSP, after: lastDjangoApp, rank: rankCSP}},
					middleware: []entry{{name: middlewareDjangoCSP, after: lastDjangoMiddleware, rank: rankCSP}},
				},
			}
		},
	}
}
