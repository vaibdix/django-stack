package scaffold

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vaibdix/django-stack/internal/config"
)

func TestMypyLoadsSettingsThatNeedNoEnv(t *testing.T) {
	t.Parallel()

	split := &Env{cfg: config.Choices{Name: "shop", AddOns: config.NewSet(config.MultiEnv)}}
	if module, err := split.at(t.TempDir()).typecheckSettingsModule(); err != nil || module != "shop.settings.test" {
		t.Errorf("the split layout should type-check against settings.test, got %q (%v)", module, err)
	}

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "shop"), 0o755); err != nil {
		t.Fatal(err)
	}
	single := &Env{cfg: config.Choices{Name: "shop"}}
	module, err := single.at(root).typecheckSettingsModule()
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
