package scaffold

import (
	"cmp"
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/vaibdix/django-stack/internal/pyenv"
)

func (e *Env) runMigrations(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, defaultCommandTimeout)
	defer cancel()

	pythonPath := pyenv.InterpreterPath(e.dir)

	if _, err := e.runIn(ctx, e.dir, pythonPath, manageFile, "migrate"); err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}
	e.Step("Applied database migrations.")
	e.Step("Database configured via .env file.")

	if e.cfg.App().Ok() {
		if _, err := e.runIn(ctx, e.dir, pythonPath, manageFile, "create_sample_data"); err != nil {
			return fmt.Errorf("create sample data: %w", err)
		}
		e.Step("Created sample data for the example app.")
	}

	return nil
}

const migrationTimestamp = "2006-01-02 15:04"

type migrationData struct {
	Django    string
	Generated string
	AppName   string
	Imports   []string
	IndexName string
}

func (p project) migrationData() migrationData {
	return migrationData{Django: p.cfg.Django().String(), Generated: p.resolved.Generated.UTC().Format(migrationTimestamp)}
}

func (p project) accountsMigration() string {
	return render("accounts_migration.py.tmpl", p.migrationData())
}

func (p project) appMigrationFiles() fileSet {
	data := p.migrationData()
	data.AppName = p.cfg.App().String()
	data.IndexName = djangoIndexName(p.cfg.App().String()+"_book", "-created_at", "-id")
	data.Imports = djangoMigrationImports("import "+p.cfg.App().String()+".fields", "import datetime", "import django.core.validators",
		"from django.db import migrations, models")
	files := fileSet{}
	files.add(filepath.Join(p.cfg.App().String(), "migrations", "0001_initial.py"), render("app_migration.py.tmpl", data), fileMode)
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

func djangoIndexName(table string, fieldsWithOrder ...string) string {
	digest := md5.New()
	for _, part := range append(append([]string{table}, fieldsWithOrder...), "idx") {
		digest.Write([]byte(part))
	}
	column := strings.TrimPrefix(fieldsWithOrder[0], "-")
	return table[:min(len(table), 11)] + "_" + column[:min(len(column), 7)] + "_" + hex.EncodeToString(digest.Sum(nil))[:6] + "_idx"
}
