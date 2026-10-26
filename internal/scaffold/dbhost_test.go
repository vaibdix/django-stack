package scaffold

import (
	"strings"
	"testing"

	"github.com/vaibdix/django-stack/internal/config"
	"github.com/vaibdix/django-stack/internal/report"
)

const socketHost = "localhost"

func hostEnvFor(db config.Database, pydantic bool) *Env {
	return &Env{
		cfg: config.Choices{
			Name:     "demo",
			Database: db,
			AddOns:   addOnsWhere(map[config.AddOn]bool{config.Pydantic: pydantic}),
		},
		resolved: credentialsFor(db, config.DatabaseCredentials{}),
		reporter: report.Nop{},
	}
}

func dbHostFromDotenv(t *testing.T, label, content string) string {
	t.Helper()

	for _, line := range strings.Split(content, "\n") {
		if value, ok := strings.CutPrefix(line, "DB_HOST="); ok {
			return value
		}
	}
	t.Fatalf("%s declares no DB_HOST:\n%s", label, content)
	return ""
}

func dbHostBetween(t *testing.T, label, content, prefix string) string {
	t.Helper()

	start := strings.Index(content, prefix)
	if start < 0 {
		t.Fatalf("%s does not declare a DB_HOST default (looked for %q):\n%s", label, prefix, content)
	}
	rest := content[start+len(prefix):]
	end := strings.Index(rest, `"`)
	if end < 0 {
		t.Fatalf("%s has an unterminated DB_HOST default:\n%s", label, content)
	}
	return rest[:end]
}

func mysqlHostsByArtifact(t *testing.T, pydantic bool) map[string]string {
	t.Helper()

	e := hostEnvFor(config.MySQLDocker, pydantic)
	hosts := map[string]string{
		".env":         dbHostFromDotenv(t, ".env", envFileFor(t, e)),
		".env.example": dbHostFromDotenv(t, ".env.example", e.renderEnvExampleContent()),
	}

	if pydantic {
		conf := render("conf.py.tmpl", e.confTemplateData())
		hosts["conf.py"] = dbHostBetween(t, "conf.py", conf, `DB_HOST: str = "`)
		return hosts
	}

	block := envConfigBlock(e.envConfig())
	hosts["settings.py"] = dbHostBetween(t, "settings.py", block, `env("DB_HOST", default="`)
	return hosts
}

func TestTheMySQLHostIsOneTheDriverDialsOverTCP(t *testing.T) {
	t.Parallel()

	for _, pydantic := range []bool{false, true} {
		name := "django-environ"
		if pydantic {
			name = "pydantic-settings"
		}

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			for artifact, host := range mysqlHostsByArtifact(t, pydantic) {
				if host == "" {
					t.Errorf("%s sets an empty DB_HOST", artifact)
					continue
				}
				if host == socketHost {
					t.Errorf("%s sets DB_HOST=%q; mysqlclient reads that as a unix socket path "+
						"and ignores DB_PORT, so the project cannot reach its own container",
						artifact, host)
				}
			}
		})
	}
}

func TestThePostgresHostIsLeftAsLocalhost(t *testing.T) {
	t.Parallel()

	for _, db := range []config.Database{config.PostgresDocker} {
		t.Run(string(db), func(t *testing.T) {
			t.Parallel()

			e := hostEnvFor(db, false)
			got := dbHostFromDotenv(t, ".env", envFileFor(t, e))

			if got != socketHost {
				t.Errorf("%s sets DB_HOST=%q, want %q; libpq dials localhost over TCP, "+
					"so only the MySQL path needs an address", db, got, socketHost)
			}
		})
	}
}
