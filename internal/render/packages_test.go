package render

import (
	"slices"
	"testing"

	"github.com/vaibdix/django-stack/internal/config"
)

func TestTheProjectVenvGetsOnlyTheFormattersItDeclares(t *testing.T) {
	t.Parallel()

	plan := (&Project{Cfg: mustSpec(config.Choices{Django: "6.1.2"})}).PackagePlan()
	for _, formatter := range []string{PkgRuff, PkgDjLint} {
		if slices.Contains(plan.All(), formatter) {
			t.Errorf("%s is installed into the project although it was not picked", formatter)
		}
	}

	plan = (&Project{Cfg: mustSpec(config.Choices{Django: "6.1.2", Stack: config.StackStandard, AddOns: config.NewSet(config.Ruff)})}).PackagePlan()
	for _, formatter := range []string{PkgRuff, PkgDjLint} {
		if !slices.Contains(plan.Dev, formatter) {
			t.Errorf("%s was picked but is not a declared dev dependency: %v", formatter, plan.Dev)
		}
	}
}

func TestTheStubsFollowTheDjangoSeries(t *testing.T) {
	t.Parallel()

	for version, want := range map[string]string{"5.2.18": ">=5.2,<5.3", "6.0.9": ">=6.0,<6.1", "6.1.2": ">=6.1,<6.2"} {
		plan := (&Project{Cfg: mustSpec(config.Choices{Django: version, AddOns: config.NewSet(config.Mypy)})}).PackagePlan()
		if !slices.Contains(plan.Dev, pkgDjangoStubs+"[compatible-mypy]"+want) {
			t.Errorf("Django %s: django-stubs is not bounded to %s: %v", version, want, plan.Dev)
		}
		if !slices.Contains(plan.Runtime, PkgDjangoStubsExt+want) {
			t.Errorf("Django %s: django-stubs-ext is not bounded to %s: %v", version, want, plan.Runtime)
		}
	}
}
