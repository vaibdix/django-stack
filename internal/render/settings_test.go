package render

import (
	"strings"
	"testing"

	"github.com/vaibdix/django-stack/internal/config"
)

func TestProductionSettingsHaveNoTopLevelConnMaxAge(t *testing.T) {
	t.Parallel()

	for _, mailers := range []bool{false, true} {
		production := Render("production.py.tmpl", MultiEnvSettingsData{ModuleName: "my_shop", Mailers: mailers, Env: environEnv().ProductionEnvExpressions()})
		for line := range strings.SplitSeq(production, "\n") {
			if strings.HasPrefix(line, "CONN_MAX_AGE") {
				t.Errorf("CONN_MAX_AGE is a key of each DATABASES entry; at module level Django ignores it:\n%s", production)
			}
		}
	}
}

func TestHealthChecksAreNotRedirectedToHTTPS(t *testing.T) {
	t.Parallel()

	production := Render("production.py.tmpl", MultiEnvSettingsData{ModuleName: "my_shop", Env: environEnv().ProductionEnvExpressions()})
	if !strings.Contains(production, `SECURE_REDIRECT_EXEMPT = [r"^healthz/$"]`) {
		t.Errorf("a load balancer probing /healthz/ over HTTP gets a 301 and marks the app down:\n%s", production)
	}
}

func TestEachDatabaseRendersExactlyOneDatabasesBlock(t *testing.T) {
	t.Parallel()

	for _, db := range config.Databases() {
		for _, addOns := range []config.Set[config.AddOn]{config.NewSet[config.AddOn](), config.NewSet(config.Pydantic), config.NewSet(config.MultiEnv)} {
			e := &Project{Cfg: mustSpec(config.Choices{Name: "myproject", Django: "6.1.2", Database: db, Stack: config.StackAPIOnly, AddOns: addOns})}
			settings := settingsText(e)
			if strings.Count(settings, "\nDATABASES = {") != 1 {
				t.Errorf("%s %v: want one DATABASES block:\n%s", db, addOns, settings)
			}
			if !strings.Contains(settings, `"ENGINE": "`+db.Spec().Engine+`"`) {
				t.Errorf("%s: the %s engine is missing:\n%s", db, db.Spec().Engine, settings)
			}
			if db.UsesDocker() && strings.Contains(settings, "sqlite3") {
				t.Errorf("%s still mentions sqlite3:\n%s", db, settings)
			}
			for _, setting := range []string{"SECRET_KEY = ", "DEBUG = "} {
				if strings.Count(settings, "\n"+setting) != 1 {
					t.Errorf("%s: want exactly one %s:\n%s", db, setting, settings)
				}
			}
		}
	}
}
