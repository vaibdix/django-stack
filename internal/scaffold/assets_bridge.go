package scaffold

import (
	"github.com/vaibdix/django-stack/internal/assets"
	"github.com/vaibdix/django-stack/internal/config"
)

func (e *Env) apiDocsRoute() string {
	if e.cfg.Has(config.Ninja) {
		return assets.NinjaDocsRoute
	}
	return assets.DRFDocsRoute
}

func (e *Env) pageOptions() assets.PageOptions {
	return assets.PageOptions{
		AppName:      e.cfg.AppName,
		Tailwind:     e.cfg.UsesTailwind(),
		API:          e.cfg.HasAPI(),
		APIDocsRoute: e.apiDocsRoute(),
		AppRoute:     assets.AppListRoute,
		HTMX:         e.cfg.UsesHTMX(),
		AllAuth:      e.cfg.Has(config.AllAuth),
	}
}

func (e *Env) hasBookPages() bool {
	return e.cfg.AppName != "" && !e.cfg.APIOnly()
}
