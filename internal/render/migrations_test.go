package render

import (
	"slices"
	"testing"
)

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
