package scaffold

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vaibdix/django-stack/internal/command/commandtest"
	"github.com/vaibdix/django-stack/internal/config"
)

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

	for _, unwanted := range []string{"django.contrib.sites", "SITE_ID"} {
		if strings.Contains(string(settings), unwanted) {
			t.Errorf("allauth settings still carry %s:\n%s", unwanted, settings)
		}
	}
}
