package scaffold

import (
	"strings"
	"testing"

	"github.com/vaibdix/django-stack/internal/config"
)

func TestTypingOnlyImportsMoveOnlyWhereAnnotationsAreLazy(t *testing.T) {
	t.Parallel()

	eager := &Env{cfg: config.Project{DjangoVersion: "6.1.2", SelectedPythonVersion: "Python 3.13.13", SetupNinja: true}}
	if commands := eager.typingImportCommands(); len(commands) != 0 {
		t.Errorf("Python 3.13 evaluates annotations eagerly, so no import may move: %v", commands)
	}

	lazy := &Env{cfg: config.Project{DjangoVersion: "6.1.2", SelectedPythonVersion: "Python 3.14.0", SetupNinja: true}}
	commands := lazy.typingImportCommands()
	if len(commands) != 1 {
		t.Fatalf("Python 3.14 should move typing-only imports in one ruff run, got %v", commands)
	}
	args := strings.Join(commands[0].Args, " ")
	for _, needle := range []string{"--select TC001,TC002,TC003", "--target-version py314", `"ninja.Schema"`, `"ninja.Router.post"`, `"pydantic.BaseModel"`} {
		if !strings.Contains(args, needle) {
			t.Errorf("the typing-only import fix lacks %s, so it could move an import ninja reads at runtime: %s", needle, args)
		}
	}
}

func TestTheRuffConfigKeepsImportsPydanticAndNinjaRead(t *testing.T) {
	t.Parallel()

	section := (&Env{cfg: config.Project{SetupNinja: true, SetupPydanticConf: true}}).runtimeEvaluatedSection()
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
