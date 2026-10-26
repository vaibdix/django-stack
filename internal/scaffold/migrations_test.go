package scaffold

import (
	"slices"
	"strings"
	"testing"

	"github.com/vaibdix/django-stack/internal/command/commandtest"
	"github.com/vaibdix/django-stack/internal/config"
)

func TestMigrationsDoNotPretendToCreateTheProductionCacheTable(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	fake := commandtest.NewFake()
	rep := &RecordingReporter{}
	e := &Env{cfg: mustSpec(config.Choices{Name: "demo"}), runner: fake, reporter: rep}

	e.dir = root
	if err := e.runMigrations(t.Context()); err != nil {
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
	for _, line := range fake.CommandLines() {
		if strings.Contains(line, "makemigrations") {
			t.Errorf("ran %q although the initial migrations are rendered with the project", line)
		}
	}
}

func TestTheIndexNameIsTheOneDjangoWouldGenerate(t *testing.T) {
	t.Parallel()

	for table, want := range map[string]string{
		"catalog_book":                   "catalog_boo_created_46e110_idx",
		"library_inventory_records_book": "library_inv_created_59dd06_idx",
	} {
		if got := djangoIndexName(table, "-created_at", "-id"); got != want {
			t.Errorf("djangoIndexName(%s) = %s, want %s", table, got, want)
		}
	}
}

func TestMigrationImportsAreOrderedAsDjangoWritesThem(t *testing.T) {
	t.Parallel()

	for app, want := range map[string][]string{
		"catalog": {"import catalog.fields", "import datetime", "import django.core.validators", "from django.db import migrations, models"},
		"library": {"import datetime", "import django.core.validators", "import library.fields", "from django.db import migrations, models"},
	} {
		got := djangoMigrationImports("from django.db import migrations, models", "import django.core.validators", "import datetime", "import "+app+".fields")
		if !slices.Equal(got, want) {
			t.Errorf("%s: imports ordered %q, want %q", app, got, want)
		}
	}
}
