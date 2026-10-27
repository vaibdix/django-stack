package scaffold

import (
	"strings"
	"testing"

	"github.com/vaibdix/django-stack/internal/config"
)

func ninjaProject(appName string) config.Choices {
	cfg := config.Choices{
		Name:     "my-shop",
		App:      appName,
		Django:   "6.0",
		Database: config.SQLite,
	}
	cfg.Stack, cfg.AddOns = config.StackAPIOnly, config.NewSet(config.Ninja)
	return cfg
}

func TestNinjaEnablesThePydanticMypyPlugin(t *testing.T) {
	t.Parallel()

	written := mypyConfigFor(t, ninjaProject("catalog"))
	if !strings.Contains(written, `"pydantic.mypy"`) {
		t.Errorf("Ninja schemas are pydantic models; without the plugin, disallow_any_explicit "+
			"rejects every Schema subclass:\n%s", written)
	}
}
