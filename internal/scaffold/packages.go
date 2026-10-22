package scaffold

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/vaibdix/django-stack/internal/config"
)

type packagePlan struct {
	runtime []string
	dev     []string
}

func (p packagePlan) all() []string {
	all := append([]string{}, p.runtime...)
	return append(all, p.dev...)
}

func (e *Env) packagePlan() packagePlan {
	cfg := e.cfg
	plan := packagePlan{
		runtime: append([]string{"django==" + cfg.DjangoVersion, pkgArgon2, pkgWhiteNoise, pkgGunicorn}, e.configPackages()...),
	}
	add := func(list *[]string, when bool, packages ...string) {
		if when {
			*list = append(*list, packages...)
		}
	}

	add(&plan.runtime, cfg.DatabaseType.UsesPostgres(), pkgPsycopg)
	add(&plan.runtime, cfg.DatabaseType == config.MySQLDocker, pkgMySQLClient)
	add(&plan.runtime, cfg.SetupHTMX, pkgHTMX)
	add(&plan.runtime, cfg.SetupAllAuth, pkgAllAuth)
	add(&plan.runtime, cfg.SetupStructlog, pkgStructlog, pkgDjangoStructlog)
	add(&plan.runtime, cfg.SetupSentry, pkgSentry)
	add(&plan.runtime, !config.DjangoSupportsNativeCSP(cfg.DjangoVersion), pkgDjangoCSP)
	add(&plan.runtime, cfg.SetupRestFramework, pkgRestFramework, pkgDRFSpectacular, pkgDRFSpectacularSidecar)
	add(&plan.runtime, cfg.SetupNinja, pkgNinja)
	stubs, stubsExt := djangoStubsSpecs(cfg.DjangoVersion)
	add(&plan.runtime, cfg.SetupMypy, stubsExt)

	add(&plan.dev, cfg.CreateTemplates, pkgBrowserReload)
	add(&plan.dev, cfg.SetupMypy, pkgMypy, stubs)
	add(&plan.dev, cfg.SetupMypy && cfg.SetupRestFramework, pkgDRFStubs)
	add(&plan.dev, cfg.SetupPreCommit, pkgPreCommit)

	plan.dev = append(plan.dev, e.declaredFormattingPackages()...)
	return plan
}

func djangoStubsSpecs(djangoVersion string) (stubs, ext string) {
	series := config.DjangoSeries(djangoVersion)
	bound := ">=" + series
	if major, minor, ok := strings.Cut(series, "."); ok {
		if next, err := strconv.Atoi(minor); err == nil {
			bound += fmt.Sprintf(",<%s.%d", major, next+1)
		}
	}
	return pkgDjangoStubs + "[compatible-mypy]" + bound, pkgDjangoStubsExt + bound
}

func (e *Env) typedAgainstStubs52() bool {
	return e.cfg.SetupMypy && config.DjangoSeries(e.cfg.DjangoVersion) == "5.2"
}
