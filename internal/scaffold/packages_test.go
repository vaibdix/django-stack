package scaffold

import (
	"slices"
	"strings"
	"testing"

	"github.com/vaibdix/django-cli/internal/config"
	"github.com/vaibdix/django-cli/internal/scaffold/command/commandtest"
)

func TestEveryPackageIsInstalledInOneResolve(t *testing.T) {
	t.Parallel()

	cfg := config.Project{ProjectName: "demo", DjangoVersion: "6.1.2", DatabaseType: config.MySQLDocker}
	cfg.ApplyFeatures(append(config.StackTailwindHTMX.Features(),
		config.FeatureRestFramework, config.FeatureAllAuth, config.FeatureMypy, config.FeaturePreCommit,
		config.FeatureStructlog, config.FeatureSentry, config.FeatureRuff))

	fake := commandtest.NewFake()
	e := &Env{Cfg: cfg, Runner: fake, Reporter: NopReporter{}}
	if err := e.installDjango(t.TempDir()); err != nil {
		t.Fatalf("installDjango: %v", err)
	}

	installs := 0
	var line string
	for _, call := range fake.CommandLines() {
		if strings.Contains(call, " install ") {
			installs++
			line = call
		}
	}
	if installs != 1 {
		t.Fatalf("%d install commands ran, want one resolve for the whole project: %v", installs, fake.CommandLines())
	}
	for _, pkg := range []string{"django==6.1.2", PkgMySQLClient, PkgHTMX, PkgAllAuth, PkgRestFramework,
		PkgDRFStubs, PkgPreCommit, PkgStructlog, PkgSentry, PkgRuff, PkgDjLint, PkgDjangoStubsExt} {
		if !strings.Contains(line, " "+pkg) {
			t.Errorf("the install does not include %s: %s", pkg, line)
		}
	}
}

func TestFormattersAreAlwaysInstalledButOnlyDeclaredWhenPicked(t *testing.T) {
	t.Parallel()

	plan := (&Env{Cfg: config.Project{DjangoVersion: "6.1.2"}}).packagePlan()
	for _, formatter := range []string{PkgRuff, PkgDjLint} {
		if !slices.Contains(plan.all(), formatter) {
			t.Errorf("%s is not installed, so the scaffold cannot format its output", formatter)
		}
		if slices.Contains(plan.dev, formatter) || slices.Contains(plan.runtime, formatter) {
			t.Errorf("%s is declared although it was not picked", formatter)
		}
	}
}

func TestInstalledVersionsAreReadOnce(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	fake := commandtest.NewFake()
	e := &Env{Cfg: config.Project{ProjectName: "demo"}, Runner: fake, Reporter: NopReporter{}}
	for range 2 {
		if _, err := e.freezeVersions(root); err != nil {
			t.Fatalf("freezeVersions: %v", err)
		}
	}

	freezes := 0
	for _, line := range fake.CommandLines() {
		if strings.Contains(line, "freeze") {
			freezes++
		}
	}
	if freezes != 1 {
		t.Errorf("pip freeze ran %d times; the manifest and the stack record need it once", freezes)
	}
}
