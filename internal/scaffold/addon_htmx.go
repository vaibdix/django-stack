package scaffold

import (
	"github.com/vaibdix/django-stack/internal/config"
)

func htmxFeature() feature {
	return feature{
		enabled: config.Choices.UsesHTMX,
		contribute: func(project) contribution {
			return contribution{
				runtime:  []string{pkgHTMX},
				settings: settingsContribution{apps: []string{appHTMX}, middleware: []string{middlewareHTMX}},
			}
		},
	}
}
