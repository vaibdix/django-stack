package scaffold

import "github.com/vaibdix/django-stack/internal/config"

func cspFeature() feature {
	return feature{
		enabled: func(c config.Choices) bool { return !config.DjangoSupportsNativeCSP(c.Django) },
		contribute: func(project) contribution {
			return contribution{runtime: []string{pkgDjangoCSP}}
		},
	}
}
