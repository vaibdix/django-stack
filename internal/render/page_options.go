package render

import (
	"github.com/vaibdix/django-stack/internal/assets"
	"github.com/vaibdix/django-stack/internal/config"
)

func (p Project) apiDocsRoute() string {
	if p.Cfg.Has(config.Ninja) {
		return assets.NinjaDocsRoute
	}
	return assets.DRFDocsRoute
}

func (p Project) pageOptions() assets.PageOptions {
	return assets.PageOptions{
		AppName:      p.Cfg.App().String(),
		Tailwind:     p.Cfg.UsesTailwind(),
		API:          p.Cfg.HasAPI(),
		APIDocsRoute: p.apiDocsRoute(),
		AppRoute:     assets.AppListRoute,
		HTMX:         p.Cfg.UsesHTMX(),
		AllAuth:      p.Cfg.Has(config.AllAuth),
	}
}

func (p Project) hasBookPages() bool {
	return p.Cfg.App().Ok() && !p.Cfg.APIOnly()
}
