package scaffold

import (
	"strings"
	"testing"

	"github.com/vaibdix/django-stack/internal/command/commandtest"
	"github.com/vaibdix/django-stack/internal/config"
	"github.com/vaibdix/django-stack/internal/port"
	"github.com/vaibdix/django-stack/internal/render"
	"github.com/vaibdix/django-stack/internal/report"
)

func TestEveryPackageIsInstalledInOneResolve(t *testing.T) {
	t.Parallel()

	cfg := config.Choices{Name: "demo", Django: "6.1.2", Database: config.MySQLDocker}
	cfg.Stack, cfg.AddOns = config.StackTailwindHTMX, config.NewSet(config.RestFramework, config.AllAuth, config.Mypy, config.PreCommit,
		config.Structlog, config.Sentry, config.Ruff)

	fake := commandtest.NewFake()
	e := &Env{Cfg: mustSpec(cfg), dir: t.TempDir(), reporter: report.Nop{}, ports: port.Ports{Run: fake}}
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
	for _, pkg := range []string{"django==6.1.2", "mysqlclient", render.PkgHTMX, render.PkgAllAuth, render.PkgRestFramework,
		render.PkgDRFStubs, render.PkgPreCommit, render.PkgStructlog, render.PkgSentry, render.PkgRuff, render.PkgDjLint, render.PkgDjangoStubsExt} {
		if !strings.Contains(line, " "+pkg) {
			t.Errorf("the install does not include %s: %s", pkg, line)
		}
	}
}

func TestInstalledVersionsAreReadOnce(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	fake := commandtest.NewFake()
	e := &Env{Cfg: mustSpec(config.Choices{Name: "demo"}), reporter: report.Nop{}, ports: port.Ports{Run: fake}}
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
