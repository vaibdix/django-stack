package scaffold

import (
	"slices"
	"strings"
	"testing"

	"github.com/vaibdix/django-stack/internal/command/commandtest"
	"github.com/vaibdix/django-stack/internal/config"
	"github.com/vaibdix/django-stack/internal/port"
	"github.com/vaibdix/django-stack/internal/report"
)

func TestEveryPackageIsInstalledInOneResolve(t *testing.T) {
	t.Parallel()

	cfg := config.Choices{Name: "demo", Django: "6.1.2", Database: config.MySQLDocker}
	cfg.Stack, cfg.AddOns = config.StackTailwindHTMX, config.NewSet(config.RestFramework, config.AllAuth, config.Mypy, config.PreCommit,
		config.Structlog, config.Sentry, config.Ruff)

	fake := commandtest.NewFake()
	e := &Env{cfg: mustSpec(cfg), dir: t.TempDir(), reporter: report.Nop{}, ports: port.Ports{Run: fake}}
	if err := e.installDjango(t.Context()); err != nil {
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

	plan := (&Env{cfg: mustSpec(config.Choices{Django: "6.1.2"})}).packagePlan()
	for _, formatter := range []string{pkgRuff, pkgDjLint} {
		if slices.Contains(plan.all(), formatter) {
			t.Errorf("%s is installed into the project although it was not picked", formatter)
		}
	}

	plan = (&Env{cfg: mustSpec(config.Choices{Django: "6.1.2", Stack: config.StackStandard, AddOns: config.NewSet(config.Ruff)})}).packagePlan()
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
	e := &Env{cfg: mustSpec(config.Choices{Name: "demo"}), reporter: report.Nop{}, ports: port.Ports{Run: fake}}
	for range 2 {
		if _, err := e.freezeVersions(t.Context(), root); err != nil {
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
		plan := (&Env{cfg: mustSpec(config.Choices{Django: version, AddOns: config.NewSet(config.Mypy)})}).packagePlan()
		if !slices.Contains(plan.dev, pkgDjangoStubs+"[compatible-mypy]"+want) {
			t.Errorf("Django %s: django-stubs is not bounded to %s: %v", version, want, plan.dev)
		}
		if !slices.Contains(plan.runtime, pkgDjangoStubsExt+want) {
			t.Errorf("Django %s: django-stubs-ext is not bounded to %s: %v", version, want, plan.runtime)
		}
	}
}
