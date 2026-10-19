package scaffold

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/vaibdix/django-cli/internal/config"
	"github.com/vaibdix/django-cli/internal/scaffold/command/commandtest"
)

func TestTheStubsFollowTheDjangoSeries(t *testing.T) {
	t.Parallel()

	for version, want := range map[string]string{"5.2.18": ">=5.2,<5.3", "6.0.9": ">=6.0,<6.1", "6.1.2": ">=6.1,<6.2"} {
		plan := (&Env{Cfg: config.Project{DjangoVersion: version, SetupMypy: true}}).packagePlan()
		if !slices.Contains(plan.dev, PkgDjangoStubs+"[compatible-mypy]"+want) {
			t.Errorf("Django %s: django-stubs is not bounded to %s: %v", version, want, plan.dev)
		}
		if !slices.Contains(plan.runtime, PkgDjangoStubsExt+want) {
			t.Errorf("Django %s: django-stubs-ext is not bounded to %s: %v", version, want, plan.runtime)
		}
	}
}

func TestTypingOnlyImportsMoveOnlyWhereAnnotationsAreLazy(t *testing.T) {
	t.Parallel()

	eager := &Env{Cfg: config.Project{DjangoVersion: "6.1.2", SelectedPythonVersion: "Python 3.13.13", SetupNinja: true}}
	if commands := eager.typingImportCommands(); len(commands) != 0 {
		t.Errorf("Python 3.13 evaluates annotations eagerly, so no import may move: %v", commands)
	}

	lazy := &Env{Cfg: config.Project{DjangoVersion: "6.1.2", SelectedPythonVersion: "Python 3.14.0", SetupNinja: true}}
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

	section := (&Env{Cfg: config.Project{SetupNinja: true, SetupPydanticConf: true}}).runtimeEvaluatedSection()
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

func TestMainJSIsAClassicScript(t *testing.T) {
	t.Parallel()

	for _, htmx := range []bool{false, true} {
		js := (&Env{Cfg: config.Project{SetupHTMX: htmx}}).mainJS()
		for _, line := range strings.Split(js, "\n") {
			if strings.HasPrefix(line, "export") || strings.HasPrefix(line, "import ") {
				t.Errorf("htmx=%v: base.html loads main.js without type=module, so %q is a SyntaxError", htmx, line)
			}
		}
	}
}

func TestMypyLoadsSettingsThatNeedNoEnv(t *testing.T) {
	t.Parallel()

	split := &Env{Cfg: config.Project{ProjectName: "shop", SetupMultiEnv: true}}
	if module, err := split.typecheckSettingsModule(t.TempDir()); err != nil || module != "shop.settings.test" {
		t.Errorf("the split layout should type-check against settings.test, got %q (%v)", module, err)
	}

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "shop"), 0o755); err != nil {
		t.Fatal(err)
	}
	single := &Env{Cfg: config.Project{ProjectName: "shop"}}
	module, err := single.typecheckSettingsModule(root)
	if err != nil || module != "shop.typecheck_settings" {
		t.Fatalf("the single-file layout should type-check against shop.typecheck_settings, got %q (%v)", module, err)
	}
	source, err := os.ReadFile(filepath.Join(root, "shop", typecheckSettingsFile))
	if err != nil {
		t.Fatal(err)
	}
	setdefault := strings.Index(string(source), "os.environ.setdefault('ENVIRONMENT', 'test')")
	if setdefault == -1 || setdefault > strings.Index(string(source), "from .settings import *") {
		t.Errorf("ENVIRONMENT=test must be set before the settings are imported:\n%s", source)
	}
}

func TestProductionRefusesTheDefaultSenderWhenUsersGetEmail(t *testing.T) {
	t.Parallel()

	for _, cfg := range []config.Project{{CreateTemplates: true}, {SetupAllAuth: true}} {
		if guard := (&Env{Cfg: cfg}).fromEmailGuard(""); !strings.Contains(guard, "if DEFAULT_FROM_EMAIL == '"+djangoDefaultFromEmail+"':") {
			t.Errorf("%+v: production would send account email from %s:\n%s", cfg, djangoDefaultFromEmail, guard)
		}
	}
	if guard := (&Env{}).fromEmailGuard(""); guard != "" {
		t.Errorf("an API-only project sends no account email, so it needs no sender:\n%s", guard)
	}
}

func TestBooleanSettingsRejectWhatTheyDoNotRecognise(t *testing.T) {
	t.Parallel()

	settings := settingsWithObservability(t, observabilityConfig())
	if strings.Contains(settings, "env.bool(") {
		t.Errorf("django-environ's env.bool turns a typo into False; booleans must go through env_bool:\n%s", settings)
	}
	if !strings.Contains(settings, "raise ImproperlyConfigured(message)") || !strings.Contains(settings, "def env_bool(") {
		t.Errorf("env_bool is missing or does not refuse an unknown value:\n%s", settings)
	}
}

func TestAllAuthLeavesTheSitesFrameworkOut(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	cfg := config.Project{ProjectName: "shop", SetupAllAuth: true}
	if err := os.MkdirAll(filepath.Join(root, cfg.ModuleName()), 0o755); err != nil {
		t.Fatal(err)
	}
	settingsPath := filepath.Join(root, cfg.ModuleName(), SettingsFile)
	if err := os.WriteFile(settingsPath, []byte(djangoAdminSettingsFixture), 0o644); err != nil {
		t.Fatal(err)
	}
	e := &Env{Cfg: cfg, Reporter: NopReporter{}, Runner: commandtest.NewFake()}
	if err := e.applyAllAuth(root); err != nil {
		t.Fatal(err)
	}
	settings, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatal(err)
	}
	// Without the sites framework allauth names the site after the request's
	// host; with it, every email says "example.com" until someone edits a row.
	for _, unwanted := range []string{"django.contrib.sites", "SITE_ID"} {
		if strings.Contains(string(settings), unwanted) {
			t.Errorf("allauth settings still carry %s:\n%s", unwanted, settings)
		}
	}
}
