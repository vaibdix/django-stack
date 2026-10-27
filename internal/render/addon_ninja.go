package render

import (
	"fmt"

	"github.com/vaibdix/django-stack/internal/config"
)

func ninjaFeature() feature {
	return feature{
		enabled: Has(config.Ninja),
		contribute: func(p Project) contribution {
			return contribution{
				runtime:  []string{pkgNinja},
				readme:   func(d *readmeData) { d.REST, d.Ninja = true, true },
				urls:     p.ninjaURLs(),
				settings: settingsContribution{apps: []entry{{name: appNinja, after: atEnd, rank: rankAPI}}},
			}
		},
	}
}

func (p Project) ninjaURLs() urlContribution {
	urls := urlContribution{home: "/api/v1/docs"}
	if !p.Cfg.App().Ok() {
		return urls
	}
	urls.imports = []string{fmt.Sprintf("from %s.api import api", p.Cfg.App().String())}
	urls.routes = []route{{rank: rankAPIRoute, line: `    path("api/v1/", api.urls),`}}
	if !p.Cfg.APIOnly() {
		urls.routes = append(urls.routes, route{rank: rankAPIRoute + 1, line: fmt.Sprintf(`    path("%s/", include("%s.urls")),`, p.Cfg.App().String(), p.Cfg.App().String())})
	}
	urls.appRoute = true
	return urls
}

type ninjaPackageData struct {
	AppName     string
	ProjectName string
}

func generateNinjaPackage(appName, projectName string) map[string]string {
	data := ninjaPackageData{AppName: appName, ProjectName: projectName}
	return map[string]string{
		"__init__.py": Render("ninja_init.py.tmpl", data),
		"api.py":      Render("ninja_api.py.tmpl", data),
		"router.py":   Render("ninja_router.py.tmpl", data),
		"schemas.py":  Render("ninja_schemas.py.tmpl", data),
	}
}
