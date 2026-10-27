package scaffold

import (
	"github.com/vaibdix/django-stack/internal/config"
)

func addOnsWhere(flags map[config.AddOn]bool) config.Set[config.AddOn] {
	set := config.NewSet[config.AddOn]()
	for addOn, on := range flags {
		if on {
			set = set.With(addOn)
		}
	}
	return set
}

func projectForStack(stack config.Stack, appName string, extra ...config.AddOn) config.Choices {
	cfg := config.Choices{Name: "my-shop", App: appName, Database: config.SQLite}
	cfg.Stack, cfg.AddOns = stack, config.NewSet(extra...)
	return cfg
}
