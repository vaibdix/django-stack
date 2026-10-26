package scaffold

import (
	"strings"
	"testing"

	"github.com/vaibdix/django-stack/internal/config"
)

func TestTheRuffConfigKeepsImportsPydanticAndNinjaRead(t *testing.T) {
	t.Parallel()

	section := (&Env{cfg: mustSpec(config.Choices{AddOns: config.NewSet(config.Ninja, config.Pydantic)})}).runtimeEvaluatedSection()
	for _, needle := range []string{
		"[tool.ruff.lint.flake8-type-checking]",
		`runtime-evaluated-base-classes = ["pydantic.BaseModel", "pydantic_settings.BaseSettings", "ninja.Schema"]`,
		`"ninja.Router.get"`, `"ninja.NinjaAPI.exception_handler"`,
	} {
		if !strings.Contains(section, needle) {
			t.Errorf("the Ruff config lacks %s:\n%s", needle, section)
		}
	}

	if section := (&Env{}).runtimeEvaluatedSection(); section != "" {
		t.Errorf("a project with neither pydantic nor ninja needs no runtime-evaluated section:\n%s", section)
	}
}

func TestTypingOnlyImportsMoveOnlyWhereAnnotationsAreLazy(t *testing.T) {
	t.Parallel()

	for _, python := range []string{"Python 3.13.2", "Python 3.14.0"} {
		e := &Env{cfg: mustSpec(config.Choices{Name: "demo", Stack: config.StackStandard, App: "catalog"}), resolved: config.Resolved{PythonVersion: python}}
		views := string(e.appFiles()["catalog/views.py"].Content)
		guarded := strings.Contains(views, "if TYPE_CHECKING:\n    from django.forms import Form")
		if guarded != e.lazyAnnotations() {
			t.Errorf("%s: Form under TYPE_CHECKING = %v, want %v:\n%s", python, guarded, e.lazyAnnotations(), views)
		}
	}
}
