package render

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/vaibdix/django-stack/internal/config"
)

type PackagePlan struct {
	Runtime []string
	Dev     []string
}

func (p PackagePlan) All() []string {
	all := append([]string{}, p.Runtime...)
	return append(all, p.Dev...)
}

func (p Project) PackagePlan() PackagePlan {
	cfg := p.Cfg
	plan := PackagePlan{
		Runtime: append([]string{"django==" + cfg.Django().String(), pkgArgon2, pkgWhiteNoise, pkgGunicorn}, p.configPackages()...),
	}
	add := func(list *[]string, when bool, packages ...string) {
		if when {
			*list = append(*list, packages...)
		}
	}

	database := cfg.Database().Spec()
	add(&plan.Runtime, database.Driver != "", database.Driver)
	add(&plan.Dev, cfg.CreateTemplates(), pkgBrowserReload)

	plan.Runtime = append(plan.Runtime, p.contributed(func(c contribution) []string { return c.runtime })...)
	plan.Dev = append(plan.Dev, p.contributed(func(c contribution) []string { return c.dev })...)

	plan.Dev = append(plan.Dev, p.DeclaredFormattingPackages()...)
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
	return pkgDjangoStubs + "[compatible-mypy]" + bound, PkgDjangoStubsExt + bound
}

func (p Project) DeclaredFormattingPackages() []string {
	var declared []string
	if p.Cfg.Has(config.Ruff) {
		declared = append(declared, PkgRuff)
	}
	if p.Cfg.CreateTemplates() {
		declared = append(declared, PkgDjLint)
	}
	return declared
}
