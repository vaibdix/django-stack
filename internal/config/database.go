package config

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

var databases = []struct {
	Database
	DatabaseSpec
}{
	{SQLite, DatabaseSpec{
		Label:        "SQLite",
		Option:       "SQLite (default)",
		Engine:       "django.db.backends.sqlite3",
		UpsertTarget: true,
	}},
	{PostgresDocker, DatabaseSpec{
		Label:        "PostgreSQL",
		Option:       "PostgreSQL (Docker)",
		Image:        "postgres:18",
		Driver:       "psycopg[binary]",
		Engine:       "django.db.backends.postgresql",
		Host:         "localhost",
		Port:         5432,
		UpsertTarget: true,
	}},
	{MySQLDocker, DatabaseSpec{
		Label:   "MySQL",
		Option:  "MySQL (Docker)",
		Image:   "mysql:9.7",
		Driver:  "mysqlclient",
		Engine:  "django.db.backends.mysql",
		Host:    "127.0.0.1",
		Port:    3306,
		Options: []string{"'charset': 'utf8mb4'", "'isolation_level': 'read committed'"},
	}},
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
