package scaffold

import (
	"github.com/vaibdix/django-stack/internal/render"
)

func appTemplateVariants(appName string) map[string]render.AppTemplateConfig {
	return map[string]render.AppTemplateConfig{
		"plain":         {AppName: appName},
		"htmx":          {AppName: appName, HTMX: true},
		"tailwind":      {AppName: appName, Tailwind: true},
		"tailwind_htmx": {AppName: appName, Tailwind: true, HTMX: true},
	}
}
