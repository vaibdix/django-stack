package scaffold

import "github.com/vaibdix/django-cli/internal/assets"

// pageOptions derives the asset layer's render options from the project config.
//
// This is the whole adapter now. It used to build a map[string]string with
// "true"/"false" string values via a boolToString helper, so a mistyped key
// silently selected the wrong template variant.
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
