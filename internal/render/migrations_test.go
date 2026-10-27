package render

import (
	"slices"
	"testing"
)

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
