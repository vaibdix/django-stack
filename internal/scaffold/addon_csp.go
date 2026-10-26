package scaffold

import "github.com/vaibdix/django-stack/internal/config"

func cspFeature() feature {
	return feature{
		enabled: func(c config.Spec) bool { return !c.Django().SupportsNativeCSP() },
		contribute: func(project) contribution {
			return contribution{runtime: []string{pkgDjangoCSP}}
		},
	}
}
