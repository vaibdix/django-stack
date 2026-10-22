package scaffold

import (
	"slices"
	"strings"
	"testing"

	"github.com/vaibdix/django-stack/internal/command/commandtest"
	"github.com/vaibdix/django-stack/internal/config"
)

func TestEveryPackageIsInstalledInOneResolve(t *testing.T) {
	t.Parallel()

	cfg := config.Project{ProjectName: "demo", DjangoVersion: "6.1.2", DatabaseType: config.MySQLDocker}
	cfg.ApplyFeatures(append(config.StackTailwindHTMX.Features(),
		config.FeatureRestFramework, config.FeatureAllAuth, config.FeatureMypy, config.FeaturePreCommit,
		config.FeatureStructlog, config.FeatureSentry, config.FeatureRuff))

	fake := commandtest.NewFake()
	e := &Env{cfg: cfg, runner: fake, reporter: NopReporter{}}
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
	for _, pkg := range []string{"django==6.1.2", "mysqlclient", pkgHTMX, pkgAllAuth, pkgRestFramework,
		pkgDRFStubs, pkgPreCommit, pkgStructlog, pkgSentry, pkgRuff, pkgDjLint, pkgDjangoStubsExt} {
		if !strings.Contains(line, " "+pkg) {
			t.Errorf("the install does not include %s: %s", pkg, line)
		}
	}
}

func TestTheProjectVenvGetsOnlyTheFormattersItDeclares(t *testing.T) {
	t.Parallel()

	plan := (&Env{cfg: config.Project{DjangoVersion: "6.1.2"}}).packagePlan()
	for _, formatter := range []string{pkgRuff, pkgDjLint} {
		if slices.Contains(plan.all(), formatter) {
			t.Errorf("%s is installed into the project although it was not picked", formatter)
		}
	}

	plan = (&Env{cfg: config.Project{DjangoVersion: "6.1.2", SetupRuff: true, CreateTemplates: true}}).packagePlan()
	for _, formatter := range []string{pkgRuff, pkgDjLint} {
		if !slices.Contains(plan.dev, formatter) {
			t.Errorf("%s was picked but is not a declared dev dependency: %v", formatter, plan.dev)
		}
	}
}

func TestInstalledVersionsAreReadOnce(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	fake := commandtest.NewFake()
	e := &Env{cfg: config.Project{ProjectName: "demo"}, runner: fake, reporter: NopReporter{}}
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

func TestTheStubsFollowTheDjangoSeries(t *testing.T) {
	t.Parallel()

	for version, want := range map[string]string{"5.2.18": ">=5.2,<5.3", "6.0.9": ">=6.0,<6.1", "6.1.2": ">=6.1,<6.2"} {
		plan := (&Env{cfg: config.Project{DjangoVersion: version, SetupMypy: true}}).packagePlan()
		if !slices.Contains(plan.dev, pkgDjangoStubs+"[compatible-mypy]"+want) {
			t.Errorf("Django %s: django-stubs is not bounded to %s: %v", version, want, plan.dev)
		}
		if !slices.Contains(plan.runtime, pkgDjangoStubsExt+want) {
			t.Errorf("Django %s: django-stubs-ext is not bounded to %s: %v", version, want, plan.runtime)
		}
	}
}
