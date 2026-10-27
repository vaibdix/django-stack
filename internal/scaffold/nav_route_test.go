package scaffold

import (
	"regexp"
	"strings"
	"testing"

	"github.com/vaibdix/django-stack/internal/assets"
	"github.com/vaibdix/django-stack/internal/config"
)

var navAppRoute = regexp.MustCompile(`(?:app_name\|add:':|'catalog:)(\w+)'`)

func TestTheNavLinksToARouteTheAppDefines(t *testing.T) {
	t.Parallel()

	for _, stack := range []config.Stack{config.StackStandard, config.StackHTMX, config.StackTailwind, config.StackTailwindHTMX} {
		for _, api := range []config.AddOn{"", config.RestFramework, config.Ninja} {
			cfg := config.Choices{Name: "demo", App: "catalog", Django: "6.1"}
			var addOns []config.AddOn
			if api != "" {
				addOns = append(addOns, api)
			}
			cfg.Stack, cfg.AddOns = stack, config.NewSet(addOns...)

			e := &project{cfg: mustSpec(cfg)}
			base := assets.RenderPage(assets.PageBase, e.pageOptions())

			match := navAppRoute.FindStringSubmatch(base)
			if match == nil {
				t.Fatalf("%s/%q: base.html has no app link:\n%s", stack, api, base)
			}
			route := match[1]

			urls := generateAppURLs(e.cfg.App().String())
			if !strings.Contains(urls, "name='"+route+"'") && !strings.Contains(urls, `name="`+route+`"`) {
				t.Errorf("%s/%q: base.html links to catalog:%s, which the app's urls.py does not define, "+
					"so every page fails with NoReverseMatch:\n%s", stack, api, route, urls)
			}
		}
	}
}
