package scaffold

import "github.com/vaibdix/django-cli/internal/config"

type packagePlan struct {
	runtime     []string
	dev         []string
	installOnly []string
}

func (p packagePlan) all() []string {
	all := append([]string{}, p.runtime...)
	all = append(all, p.dev...)
	return append(all, p.installOnly...)
}

func (e *Env) packagePlan() packagePlan {
	cfg := e.Cfg
	plan := packagePlan{
		runtime: append([]string{"django==" + cfg.DjangoVersion, PkgArgon2, PkgWhiteNoise, PkgGunicorn}, e.configPackages()...),
		dev:     []string{PkgBrowserReload},
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
	add(&plan.runtime, cfg.SetupRestFramework, PkgRestFramework, PkgDRFSpectacular)
	add(&plan.runtime, cfg.SetupNinja, PkgNinja)
	add(&plan.runtime, cfg.SetupMypy, PkgDjangoStubsExt)

	add(&plan.dev, cfg.SetupMypy, PkgMypy, PkgDjangoStubs)
	add(&plan.dev, cfg.SetupMypy && cfg.SetupRestFramework, PkgDRFStubs)
	add(&plan.dev, cfg.SetupPreCommit, PkgPreCommit)

	declared := e.declaredFormattingPackages()
	plan.dev = append(plan.dev, declared...)
	for _, formatter := range scaffoldFormattingPackages() {
		if !containsSpec(declared, formatter) {
			plan.installOnly = append(plan.installOnly, formatter)
		}
	}
	return plan
}
