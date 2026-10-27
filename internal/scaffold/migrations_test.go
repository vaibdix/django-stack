package scaffold

import (
	"strings"
	"testing"

	"github.com/vaibdix/django-stack/internal/command/commandtest"
	"github.com/vaibdix/django-stack/internal/config"
	"github.com/vaibdix/django-stack/internal/port"
)

func TestMigrationsDoNotPretendToCreateTheProductionCacheTable(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	fake := commandtest.NewFake()
	e := &Env{Cfg: mustSpec(config.Choices{Name: "demo"}), ports: port.Ports{Run: fake}}
	messages := recordMessages(e)

	e.dir = root
	if err := e.runMigrations(t.Context()); err != nil {
		t.Fatalf("runMigrations: %v", err)
	}
	for _, line := range fake.CommandLines() {
		if strings.Contains(line, "createcachetable") {
			t.Errorf("ran %q under the development settings, which configure no database cache", line)
		}
	}
	if reported := messages(); strings.Contains(strings.Join(reported, "\n"), "cache table") {
		t.Errorf("reported a cache table that was never created: %q", reported)
	}
	for _, line := range fake.CommandLines() {
		if strings.Contains(line, "makemigrations") {
			t.Errorf("ran %q although the initial migrations are rendered with the project", line)
		}
	}
}
