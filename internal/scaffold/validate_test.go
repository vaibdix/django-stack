package scaffold

import (
	"testing"
	"time"

	"github.com/vaibdix/django-stack/internal/config"
)

func reviewClock() time.Time { return time.Date(2026, 10, 22, 0, 0, 0, 0, time.UTC) }

func specOf(t testing.TB, cfg config.Choices) config.Spec {
	t.Helper()
	spec, err := config.Parse(cfg, config.Interpreter{}, reviewClock())
	if err != nil {
		t.Fatalf("config.Parse(%+v): %v", cfg, err)
	}
	return spec
}
