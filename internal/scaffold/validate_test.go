package scaffold

import (
	"fmt"
	"testing"
	"time"

	"github.com/vaibdix/django-stack/internal/config"
)

func reviewClock() time.Time { return time.Date(2026, 10, 22, 0, 0, 0, 0, time.UTC) }

func specOf(t testing.TB, cfg config.Choices) config.Spec {
	t.Helper()
	spec, _, err := config.Parse(cfg, config.Interpreter{}, reviewClock())
	if err != nil {
		t.Fatalf("config.Parse(%+v): %v", cfg, err)
	}
	return spec
}

func mustSpec(cfg config.Choices) config.Spec {
	if cfg.Name == "" {
		cfg.Name = "demo"
	}
	if cfg.Django == "" {
		cfg.Django = "6.1.2"
	}
	if cfg.Database == "" {
		cfg.Database = config.SQLite
	}
	if cfg.Stack == "" {
		cfg.Stack = config.StackAPIOnly
	}
	if cfg.App == "" && cfg.NeedsAppName() {
		cfg.App = "catalog"
	}
	spec, _, err := config.Parse(cfg, config.Interpreter{}, reviewClock())
	if err != nil {
		panic(fmt.Sprintf("config.Parse(%+v): %v", cfg, err))
	}
	return spec
}

func withAddOns(spec config.Spec, addOns ...config.AddOn) config.Spec {
	return changed(spec, func(c *config.Choices) { c.AddOns = c.AddOns.With(addOns...) })
}

func changed(spec config.Spec, change func(*config.Choices)) config.Spec {
	cfg := spec.Choices()
	change(&cfg)
	return mustSpec(cfg)
}
