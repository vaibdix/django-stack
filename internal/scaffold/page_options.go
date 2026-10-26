package scaffold

import (
	"github.com/vaibdix/django-stack/internal/assets"
	"github.com/vaibdix/django-stack/internal/config"
)

func (p project) apiDocsRoute() string {
	if p.cfg.Has(config.Ninja) {
		return assets.NinjaDocsRoute
	}
	return assets.DRFDocsRoute
}

func (p project) pageOptions() assets.PageOptions {
	return assets.PageOptions{
		AppName:      p.cfg.App().String(),
		Tailwind:     p.cfg.UsesTailwind(),
		API:          p.cfg.HasAPI(),
		APIDocsRoute: p.apiDocsRoute(),
		AppRoute:     assets.AppListRoute,
		HTMX:         p.cfg.UsesHTMX(),
		AllAuth:      p.cfg.Has(config.AllAuth),
	}
}

func (p project) hasBookPages() bool {
	return p.cfg.App().Ok() && !p.cfg.APIOnly()
}
