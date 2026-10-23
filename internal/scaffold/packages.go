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
		runtime: append([]string{"django==" + cfg.Django, pkgArgon2, pkgWhiteNoise, pkgGunicorn}, e.configPackages()...),
	}
	add := func(list *[]string, when bool, packages ...string) {
		if when {
			*list = append(*list, packages...)
		}
	}

	database := cfg.Database.Spec()
	add(&plan.runtime, database.Driver != "", database.Driver)
	add(&plan.runtime, cfg.UsesHTMX(), pkgHTMX)
	add(&plan.runtime, cfg.Has(config.AllAuth), pkgAllAuth)
	add(&plan.runtime, cfg.Has(config.Structlog), pkgStructlog, pkgDjangoStructlog)
	add(&plan.runtime, cfg.Has(config.Sentry), pkgSentry)
	add(&plan.runtime, !config.DjangoSupportsNativeCSP(cfg.Django), pkgDjangoCSP)
	add(&plan.runtime, cfg.Has(config.RestFramework), pkgRestFramework, pkgDRFSpectacular, pkgDRFSpectacularSidecar)
	add(&plan.runtime, cfg.Has(config.Ninja), pkgNinja)
	stubs, stubsExt := djangoStubsSpecs(cfg.Django)
	add(&plan.runtime, cfg.Has(config.Mypy), stubsExt)

	add(&plan.dev, cfg.CreateTemplates(), pkgBrowserReload)
	add(&plan.dev, cfg.Has(config.Mypy), pkgMypy, stubs)
	add(&plan.dev, cfg.Has(config.Mypy) && cfg.Has(config.RestFramework), pkgDRFStubs)
	add(&plan.dev, cfg.Has(config.PreCommit), pkgPreCommit)

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
	return e.cfg.Has(config.Mypy) && config.DjangoSeries(e.cfg.Django) == "5.2"
}
