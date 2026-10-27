package render

import (
	"slices"
	"strings"
	"testing"

	"github.com/vaibdix/django-stack/internal/config"
)

func securitySettingsFor(t *testing.T, pydantic bool) string {
	t.Helper()

	cfg := config.Choices{Name: "demo", Django: "6.1.2", Database: config.SQLite, AddOns: addOnsWhere(map[config.AddOn]bool{config.Pydantic: pydantic})}
	return settingsText((&Project{Cfg: mustSpec(cfg)}))
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
		e := &Project{Cfg: mustSpec(config.Choices{AddOns: addOnsWhere(map[config.AddOn]bool{config.Pydantic: pydantic})})}
		production := Render("production.py.tmpl", MultiEnvSettingsData{Env: e.ProductionEnvExpressions()})
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

	for _, stack := range []config.Stack{config.StackStandard, config.StackAPIOnly} {
		settings := settingsText((&Project{Cfg: mustSpec(config.Choices{Name: "demo", Django: "6.1.2", Database: config.SQLite, Stack: stack, AddOns: config.NewSet(config.AllAuth)})}))
		if strings.Count(settings, "\nLOGIN_URL = ") != 1 || !strings.Contains(settings, `LOGIN_URL = "account_login"`) {
			t.Errorf("%s: LOGIN_URL is not set exactly once:\n%s", stack, settings)
		}
	}
}

func TestDjangoWithoutNativeCSPGetsTheSamePolicyFromDjangoCSP(t *testing.T) {
	t.Parallel()

	settings := settingsText((&Project{Cfg: mustSpec(config.Choices{Name: "demo", Django: "5.2.18", Database: config.SQLite, Stack: config.StackStandard})}))
	for _, want := range []string{"CONTENT_SECURITY_POLICY_REPORT_ONLY: dict[str, dict[str, list[str]]] = {", `"DIRECTIVES": {`, `"script-src": [SELF],`, "from csp.constants import SELF", `"csp",`, `"` + middlewareDjangoCSP + `"`} {
		if !strings.Contains(settings, want) {
			t.Errorf("Django 5.2 settings are missing %q, so the LTS track ships no CSP:\n%s", want, settings)
		}
	}

	e := &Project{Cfg: mustSpec(config.Choices{Django: "5.2.18"})}
	if !slices.Contains(e.PackagePlan().Runtime, pkgDjangoCSP) {
		t.Errorf("django-csp is configured but not installed: %v", e.PackagePlan().Runtime)
	}
	e.Cfg = changed(e.Cfg, func(c *config.Choices) { c.Django = "6.1.2" })
	if slices.Contains(e.PackagePlan().Runtime, pkgDjangoCSP) {
		t.Errorf("Django 6 has native CSP; django-csp would be a second policy: %v", e.PackagePlan().Runtime)
	}
}
