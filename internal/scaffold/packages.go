package scaffold

import "github.com/vaibdix/django-cli/internal/config"

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
	add(&plan.runtime, cfg.SetupMypy, PkgDjangoStubsExt)

	add(&plan.dev, cfg.CreateTemplates, PkgBrowserReload)
	add(&plan.dev, cfg.SetupMypy, PkgMypy, PkgDjangoStubs)
	add(&plan.dev, cfg.SetupMypy && cfg.SetupRestFramework, PkgDRFStubs)
	add(&plan.dev, cfg.SetupPreCommit, PkgPreCommit)

	plan.dev = append(plan.dev, e.declaredFormattingPackages()...)
	return plan
}
