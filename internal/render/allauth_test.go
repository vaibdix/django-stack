package render

import (
	"strings"
	"testing"

	"github.com/vaibdix/django-stack/internal/config"
)

func TestAllAuthLeavesTheSitesFrameworkOut(t *testing.T) {
	t.Parallel()

	settings := settingsText((&Project{Cfg: mustSpec(config.Choices{Name: "shop", Django: "6.1.2", AddOns: config.NewSet(config.AllAuth)})}))
	for _, unwanted := range []string{"django.contrib.sites", "SITE_ID"} {
		if strings.Contains(settings, unwanted) {
			t.Errorf("allauth settings still carry %s:\n%s", unwanted, settings)
		}
	}
}
