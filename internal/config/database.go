package config

import "github.com/vaibdix/django-stack/internal/catalog"

type Database string

const (
	SQLite Database = "sqlite"

	PostgresDocker Database = "postgres-docker"

	MySQLDocker Database = "mysql-docker"
)

type DatabaseSpec struct {
	Label        string
	Option       string
	Image        string
	Driver       string
	Engine       string
	Host         string
	Port         int
	Options      []string
	UpsertTarget bool
}

type databaseRow struct {
	Database
	DatabaseSpec
}

var databases = databasesFrom(catalog.Load())

func databasesFrom(c catalog.Catalog) []databaseRow {
	rows := make([]databaseRow, len(c.Databases))
	for i, row := range c.Databases {
		rows[i] = databaseRow{Database(row.Name), DatabaseSpec{
			Label: row.Label, Option: row.Option, Image: row.Image, Driver: row.Driver, Engine: row.Engine,
			Host: row.Host, Port: row.Port, Options: row.Options, UpsertTarget: row.UpsertTarget,
		}}
	}
	return rows
}

func Databases() []Database {
	all := make([]Database, 0, len(databases))
	for _, row := range databases {
		all = append(all, row.Database)
	}
	return all
}

func (d Database) lookup() (DatabaseSpec, bool) {
	for _, row := range databases {
		if row.Database == d {
			return row.DatabaseSpec, true
		}
	}
	return DatabaseSpec{}, false
}

func (d Database) Valid() bool {
	_, ok := d.lookup()
	return ok
}

func (d Database) Spec() DatabaseSpec {
	spec, _ := d.lookup()
	return spec
}

func (d Database) UsesDocker() bool { return d.Spec().Image != "" }

func (d Database) SupportsUpsertTarget() bool { return d.Spec().UpsertTarget }
