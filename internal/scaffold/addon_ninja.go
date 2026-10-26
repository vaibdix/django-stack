package scaffold

import (
	"context"
	"fmt"

	"github.com/vaibdix/django-stack/internal/config"
)

func (e *Env) applyNinja(ctx context.Context) error {
	if e.cfg.App != "" {
		if err := e.writeAPIPackage(); err != nil {
			return err
		}
	}

	return nil
}

func ninjaAddOn() addOn {
	return addOn{
		enabled: has(config.Ninja),
		steps:   []phasedStep{{phase: phaseAPI, step: step{name: "setup django ninja", label: "Setting up Django Ninja...", run: (*Env).applyNinja}}},
		contribute: func(p project) contribution {
			return contribution{
				runtime:  []string{pkgNinja},
				readme:   func(d *readmeData) { d.REST, d.Ninja = true, true },
				urls:     p.ninjaURLs(),
				settings: settingsContribution{lateApps: []string{appNinja}},
			}
		},
	}
}

func (p project) ninjaURLs() urlContribution {
	urls := urlContribution{home: "/api/v1/docs"}
	if p.cfg.App == "" {
		return urls
	}
	urls.imports = []string{fmt.Sprintf("from %s.api import api", p.cfg.App)}
	urls.patterns = []string{`    path("api/v1/", api.urls),`}
	if !p.cfg.APIOnly() {
		urls.patterns = append(urls.patterns, fmt.Sprintf(`    path("%s/", include("%s.urls")),`, p.cfg.App, p.cfg.App))
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
		"__init__.py": render("ninja_init.py.tmpl", data),
		"api.py":      render("ninja_api.py.tmpl", data),
		"router.py":   render("ninja_router.py.tmpl", data),
		"schemas.py":  render("ninja_schemas.py.tmpl", data),
	}
}
