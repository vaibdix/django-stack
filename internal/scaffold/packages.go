package scaffold

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/vaibdix/django-cli/internal/config"
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
	cfg := e.Cfg
	plan := packagePlan{
		runtime: append([]string{"django==" + cfg.DjangoVersion, PkgArgon2, PkgWhiteNoise, PkgGunicorn}, e.configPackages()...),
	}
	add := func(list *[]string, when bool, packages ...string) {
		if when {
			*list = append(*list, packages...)
		}
	}

	add(&plan.runtime, cfg.DatabaseType.UsesPostgres(), PkgPsycopg)
	add(&plan.runtime, cfg.DatabaseType == config.MySQLDocker, PkgMySQLClient)
	add(&plan.runtime, cfg.SetupHTMX, PkgHTMX)
	add(&plan.runtime, cfg.SetupAllAuth, PkgAllAuth)
	add(&plan.runtime, cfg.SetupStructlog, PkgStructlog, PkgDjangoStructlog)
	add(&plan.runtime, cfg.SetupSentry, PkgSentry)
	add(&plan.runtime, !config.DjangoSupportsNativeCSP(cfg.DjangoVersion), PkgDjangoCSP)
	add(&plan.runtime, cfg.SetupRestFramework, PkgRestFramework, PkgDRFSpectacular, PkgDRFSpectacularSidecar)
	add(&plan.runtime, cfg.SetupNinja, PkgNinja)
	stubs, stubsExt := djangoStubsSpecs(cfg.DjangoVersion)
	add(&plan.runtime, cfg.SetupMypy, stubsExt)

	add(&plan.dev, cfg.CreateTemplates, PkgBrowserReload)
	add(&plan.dev, cfg.SetupMypy, PkgMypy, stubs)
	add(&plan.dev, cfg.SetupMypy && cfg.SetupRestFramework, PkgDRFStubs)
	add(&plan.dev, cfg.SetupPreCommit, PkgPreCommit)

	plan.dev = append(plan.dev, e.declaredFormattingPackages()...)
	return plan
}

// djangoStubsSpecs pins django-stubs and django-stubs-ext to the Django series
// being generated. Each stubs release describes one series (5.2.x for Django
// 5.2, 6.1.x for 6.1), so the newest release would let mypy accept APIs the
// installed Django does not have. The compatible-mypy extra lets the resolver
// pick the mypy release that series' plugin supports.
func djangoStubsSpecs(djangoVersion string) (stubs, ext string) {
	series := config.DjangoSeries(djangoVersion)
	bound := ">=" + series
	if major, minor, ok := strings.Cut(series, "."); ok {
		if next, err := strconv.Atoi(minor); err == nil {
			bound += fmt.Sprintf(",<%s.%d", major, next+1)
		}
	}
	return PkgDjangoStubs + "[compatible-mypy]" + bound, PkgDjangoStubsExt + bound
}

// typedAgainstStubs52 reports whether mypy checks the project with the
// django-stubs 5.2 series. Those stubs leave AbstractUser.Meta undeclared and
// bound DeleteView's form to a ModelForm, where Django itself uses a plain
// Form; the 6.0 series corrected both. The generated code keeps Django's
// documented form and ignores exactly those two errors, which mypy reports
// as unused once the stubs are upgraded.
func (e *Env) typedAgainstStubs52() bool {
	return e.Cfg.SetupMypy && config.DjangoSeries(e.Cfg.DjangoVersion) == "5.2"
}
