package scaffold

import (
	"strings"
	"testing"

	"github.com/vaibdix/django-stack/internal/config"
)

func TestMypyLoadsSettingsThatNeedNoEnv(t *testing.T) {
	t.Parallel()

	split := &Env{cfg: mustSpec(config.Choices{Name: "shop", AddOns: config.NewSet(config.Mypy, config.MultiEnv)})}
	if module := split.typecheckSettingsName(); module != "shop.settings.test" {
		t.Errorf("the split layout should type-check against settings.test, got %q", module)
	}
	if len(split.typecheckFiles()) != 0 {
		t.Errorf("the split layout writes no typecheck settings module")
	}

	single := &Env{cfg: mustSpec(config.Choices{Name: "shop", AddOns: config.NewSet(config.Mypy)})}
	if module := single.typecheckSettingsName(); module != "shop.typecheck_settings" {
		t.Fatalf("the single-file layout should type-check against shop.typecheck_settings, got %q", module)
	}
	source := string(single.typecheckFiles()["shop/"+typecheckSettingsFile].Content)
	setdefault := strings.Index(source, `os.environ.setdefault("ENVIRONMENT", "test")`)
	if setdefault == -1 || setdefault > strings.Index(source, "from .settings import *") {
		t.Errorf("ENVIRONMENT=test must be set before the settings are imported:\n%s", source)
	}
}
