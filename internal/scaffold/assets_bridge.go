package scaffold

import "github.com/vaibdix/django-stack/internal/assets"

func (e *Env) apiDocsRoute() string {
	if e.cfg.SetupNinja {
		return assets.NinjaDocsRoute
	}
	return assets.DRFDocsRoute
}

func (e *Env) pageOptions() assets.PageOptions {
	return assets.PageOptions{
		AppName:      e.cfg.AppName,
		Tailwind:     e.cfg.SetupTailwind,
		API:          e.cfg.HasAPI(),
		APIDocsRoute: e.apiDocsRoute(),
		AppRoute:     assets.AppListRoute,
		HTMX:         e.cfg.SetupHTMX,
		AllAuth:      e.cfg.SetupAllAuth,
	}
}

func (e *Env) hasBookPages() bool {
	return e.cfg.AppName != "" && !e.cfg.APIOnly()
}
