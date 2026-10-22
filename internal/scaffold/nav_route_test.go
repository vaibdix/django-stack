package scaffold

import (
	"regexp"
	"strings"
	"testing"

	"github.com/vaibdix/django-cli/internal/assets"
	"github.com/vaibdix/django-cli/internal/config"
)

var navAppRoute = regexp.MustCompile(`(?:app_name\|add:':|'catalog:)(\w+)'`)

func TestTheNavLinksToARouteTheAppDefines(t *testing.T) {
	t.Parallel()

	for _, stack := range []config.Stack{config.StackStandard, config.StackHTMX, config.StackTailwind, config.StackTailwindHTMX} {
		for _, api := range []config.Feature{"", config.FeatureRestFramework, config.FeatureNinja} {
			cfg := config.Project{ProjectName: "demo", AppName: "catalog", DjangoVersion: "6.1"}
			features := stack.Features()
			if api != "" {
				features = append(features, api)
			}
			cfg.ApplyFeatures(features)

			e := &Env{Cfg: cfg, Reporter: NopReporter{}}
			base := assets.RenderPage(assets.PageBase, e.pageOptions())

			match := navAppRoute.FindStringSubmatch(base)
			if match == nil {
				t.Fatalf("%s/%q: base.html has no app link:\n%s", stack, api, base)
			}
			route := match[1]

			urls := generateAppURLs(e.Cfg.AppName)
			if !strings.Contains(urls, "name='"+route+"'") && !strings.Contains(urls, `name="`+route+`"`) {
				t.Errorf("%s/%q: base.html links to catalog:%s, which the app's urls.py does not define, "+
					"so every page fails with NoReverseMatch:\n%s", stack, api, route, urls)
			}
		}
	}
}
