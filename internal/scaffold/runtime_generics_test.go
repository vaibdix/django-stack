package scaffold

import (
	"strings"
	"testing"

	"github.com/vaibdix/django-stack/internal/config"
	"github.com/vaibdix/django-stack/internal/render"
)

func TestRuntimePatchLandsInSettingsAndSurvivesTheSplit(t *testing.T) {
	t.Parallel()

	cfg := config.Choices{
		Name:     "demo",
		Django:   "6.1",
		Database: config.SQLite,
		AddOns:   config.NewSet(config.Mypy),
	}

	written := mypyConfigFor(t, cfg)
	if !strings.Contains(written, "strict = true") {
		t.Errorf("the generated mypy config is not strict:\n%s", written)
	}
	if !strings.Contains(written, "disallow_any_explicit = true") {
		t.Errorf("the generated mypy config still permits Any:\n%s", written)
	}

	settings := settingsWithRuntimeGenerics(t, cfg)
	if !strings.Contains(settings, "django_stubs_ext.monkeypatch()") {
		t.Fatalf("settings do not install the runtime patch:\n%s", settings)
	}

	e := &render.Project{Cfg: mustSpec(cfg), Resolved: python313}
	if base := basePyFor(t, e, settings); !strings.Contains(base, "django_stubs_ext.monkeypatch()") {
		t.Errorf("the multi-env split dropped the runtime patch, so forms would stop importing:\n%s", base)
	}
}

func TestMypyOverridesCoverBothSettingsLayouts(t *testing.T) {
	t.Parallel()

	written := mypyConfigFor(t, config.Choices{
		Name:   "demo",
		Django: "6.1",
	})

	if !strings.Contains(written, `module = ["*.settings", "*.settings.*"]`) {
		t.Errorf("the settings override does not cover both layouts:\n%s", written)
	}
	if !strings.Contains(written, `module = ["*.migrations.*"]`) {
		t.Errorf("generated migrations are not exempt, so Django's own output fails the check:\n%s", written)
	}
}
