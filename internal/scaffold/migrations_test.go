package scaffold

import (
	"strings"
	"testing"

	"github.com/vaibdix/django-cli/internal/config"
	"github.com/vaibdix/django-cli/internal/scaffold/command/commandtest"
)

func TestMigrationsDoNotPretendToCreateTheProductionCacheTable(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	fake := commandtest.NewFake()
	rep := &RecordingReporter{}
	e := &Env{Cfg: config.Project{ProjectName: "demo"}, Runner: fake, Reporter: rep}

	if err := e.runMigrations(root); err != nil {
		t.Fatalf("runMigrations: %v", err)
	}
	for _, line := range fake.CommandLines() {
		if strings.Contains(line, "createcachetable") {
			t.Errorf("ran %q under the development settings, which configure no database cache", line)
		}
	}
	if strings.Contains(strings.Join(rep.Steps, "\n"), "cache table") {
		t.Errorf("reported a cache table that was never created: %q", rep.Steps)
	}
}
