package scaffold

import "github.com/vaibdix/django-stack/internal/config"

func cspAddOn() addOn {
	return addOn{
		enabled: func(c config.Choices) bool { return !config.DjangoSupportsNativeCSP(c.Django) },
		contribute: func(*Env) contribution {
			return contribution{runtime: []string{pkgDjangoCSP}}
		},
	}
}
