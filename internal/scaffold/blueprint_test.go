package scaffold

import (
	"github.com/vaibdix/django-stack/internal/config"
)

func everyAddOnBut(skip config.AddOn) config.Set[config.AddOn] {
	set := config.NewSet[config.AddOn]()
	for _, addOn := range config.AddOns() {
		if addOn != skip {
			set = set.With(addOn)
		}
	}
	return set
}
