package scaffold

import "github.com/vaibdix/django-cli/internal/assets"

func (e *Env) pageOptions() assets.PageOptions {
	return assets.PageOptions{
		AppName:         e.Cfg.AppName,
		Tailwind:        e.Cfg.SetupTailwind,
		API:             e.Cfg.SetupRestFramework,
		HTMX:            e.Cfg.SetupHTMX,
		AllAuth:         e.Cfg.SetupAllAuth,
		GlobalTemplates: e.Cfg.UseGlobalTemplates,
	}
}
