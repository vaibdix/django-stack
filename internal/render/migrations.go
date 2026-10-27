package render

import (
	"cmp"
	"path/filepath"
	"slices"
	"strings"
)

const migrationTimestamp = "2006-01-02 15:04"

type migrationData struct {
	Django    string
	Generated string
	AppName   string
	Imports   []string
}

func (p Project) migrationData() migrationData {
	return migrationData{Django: p.Cfg.Django().String(), Generated: p.Resolved.Generated.UTC().Format(migrationTimestamp)}
}

func (p Project) accountsMigration() string {
	return Render("accounts_migration.py.tmpl", p.migrationData())
}

func (p Project) appMigrationFiles() FileSet {
	data := p.migrationData()
	data.AppName = p.Cfg.App().String()
	data.Imports = djangoMigrationImports("import "+p.Cfg.App().String()+".fields", "import datetime", "import django.core.validators",
		"from django.db import migrations, models")
	files := FileSet{}
	files.add(filepath.Join(p.Cfg.App().String(), "migrations", "0001_initial.py"), Render("app_migration.py.tmpl", data), FileMode)
	return files
}

func djangoMigrationImports(imports ...string) []string {
	return slices.SortedFunc(slices.Values(imports), func(a, b string) int {
		fromA, fromB := strings.HasPrefix(a, "from "), strings.HasPrefix(b, "from ")
		if fromA != fromB {
			if fromA {
				return 1
			}
			return -1
		}
		return cmp.Compare(strings.Fields(a)[1], strings.Fields(b)[1])
	})
}
