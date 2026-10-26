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

func (p project) packagePlan() packagePlan {
	cfg := p.cfg
	plan := packagePlan{
		runtime: append([]string{"django==" + cfg.Django().String(), pkgArgon2, pkgWhiteNoise, pkgGunicorn}, p.configPackages()...),
	}
	add := func(list *[]string, when bool, packages ...string) {
		if when {
			*list = append(*list, packages...)
		}
	}

	database := cfg.Database().Spec()
	add(&plan.runtime, database.Driver != "", database.Driver)
	add(&plan.dev, cfg.CreateTemplates(), pkgBrowserReload)

	plan.runtime = append(plan.runtime, p.contributed(func(c contribution) []string { return c.runtime })...)
	plan.dev = append(plan.dev, p.contributed(func(c contribution) []string { return c.dev })...)

	plan.dev = append(plan.dev, p.declaredFormattingPackages()...)
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

func (p project) declaredFormattingPackages() []string {
	var declared []string
	if p.cfg.Has(config.Ruff) {
		declared = append(declared, pkgRuff)
	}
	if p.cfg.CreateTemplates() {
		declared = append(declared, pkgDjLint)
	}
	return declared
}
