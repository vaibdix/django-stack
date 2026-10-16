package scaffold

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/vaibdix/django-cli/internal/assets"
	"github.com/vaibdix/django-cli/internal/config"
)

var navAppRoute = regexp.MustCompile(`(?:app_name\|add:':|'catalog:)(\w+)'`)

func appURLsFor(t *testing.T, e *Env) string {
	t.Helper()

	if e.hasBookPages() {
		return generateAppURLs(e.Cfg.AppName)
	}

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, e.Cfg.AppName), 0o755); err != nil {
		t.Fatalf("creating the app directory: %v", err)
	}
	if err := e.applyAppTemplates(root); err != nil {
		t.Fatalf("writing the app templates: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(root, e.Cfg.AppName, URLsFile))
	if err != nil {
		t.Fatalf("reading the app urls: %v", err)
	}
	return string(data)
}

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
			cfg.UseGlobalTemplates = true

			e := &Env{Cfg: cfg, Reporter: NopReporter{}}
			base := assets.RenderPage(assets.PageBase, e.pageOptions())

			match := navAppRoute.FindStringSubmatch(base)
			if match == nil {
				t.Fatalf("%s/%q: base.html has no app link:\n%s", stack, api, base)
			}
			route := match[1]

			urls := appURLsFor(t, e)
			if !strings.Contains(urls, "name='"+route+"'") && !strings.Contains(urls, `name="`+route+`"`) {
				t.Errorf("%s/%q: base.html links to catalog:%s, which the app's urls.py does not define, "+
					"so every page fails with NoReverseMatch:\n%s", stack, api, route, urls)
			}
		}
	}
}
