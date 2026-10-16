package scaffold

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vaibdix/django-cli/internal/config"
)

func securitySettingsFor(t *testing.T, pydantic bool) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), SettingsFile)
	if err := os.WriteFile(path, []byte(djangoAdminSettingsFixture), 0o644); err != nil {
		t.Fatalf("writing the settings fixture: %v", err)
	}
	se, err := NewSettingsEditor(path)
	if err != nil {
		t.Fatalf("opening settings: %v", err)
	}
	cfg := EnvConfig{Database: config.SQLite, Pydantic: pydantic, ModuleName: "demo"}
	se.InjectEnvConfig(cfg)
	if err := se.InjectSecuritySettings(true, cfg); err != nil {
		t.Fatalf("InjectSecuritySettings: %v", err)
	}
	return se.content
}

func TestSecuritySettingsAreSafeByDefault(t *testing.T) {
	t.Parallel()

	for _, pydantic := range []bool{false, true} {
		settings := securitySettingsFor(t, pydantic)

		if strings.Contains(settings, "django-insecure") {
			t.Errorf("pydantic=%v: SECRET_KEY falls back to a published key:\n%s", pydantic, settings)
		}
		if strings.Contains(settings, "SECURE_HSTS_PRELOAD = True") || strings.Contains(settings, "SECURE_HSTS_INCLUDE_SUBDOMAINS = True") {
			t.Errorf("pydantic=%v: HSTS preload or subdomains is forced on:\n%s", pydantic, settings)
		}
		proxy := strings.Index(settings, "SECURE_PROXY_SSL_HEADER")
		gate := strings.Index(settings, "TRUST_PROXY_SSL_HEADER")
		if proxy < 0 || gate < 0 || gate > proxy {
			t.Errorf("pydantic=%v: the proxy header is trusted without TRUST_PROXY_SSL_HEADER:\n%s", pydantic, settings)
		}
	}
}

func TestProductionSettingsLeaveHSTSPreloadToTheOperator(t *testing.T) {
	t.Parallel()

	for _, pydantic := range []bool{false, true} {
		e := &Env{Cfg: config.Project{SetupPydanticConf: pydantic}}
		production := render("production.py.tmpl", multiEnvSettingsData{Env: e.productionEnvExpressions()})
		if strings.Contains(production, "SECURE_HSTS_PRELOAD = True") {
			t.Errorf("pydantic=%v: production.py forces HSTS preload:\n%s", pydantic, production)
		}
		if !strings.Contains(production, "SECURE_HSTS_PRELOAD") {
			t.Errorf("pydantic=%v: production.py no longer sets HSTS preload at all", pydantic)
		}
	}
}

func TestAllAuthReplacesTheLoginURLInsteadOfAddingASecondOne(t *testing.T) {
	t.Parallel()

	se := &SettingsEditor{content: "DEBUG = True\n\nLOGIN_URL = 'admin:login'\n\nX = 1\n"}
	se.SetTopLevel("LOGIN_URL", "'account_login'")
	if strings.Count(se.content, "LOGIN_URL") != 1 || !strings.Contains(se.content, "LOGIN_URL = 'account_login'") {
		t.Errorf("LOGIN_URL is not set exactly once:\n%s", se.content)
	}
}
