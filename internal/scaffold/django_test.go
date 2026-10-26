package scaffold

import (
	"strings"
	"testing"

	"github.com/vaibdix/django-stack/internal/config"
)

func TestProductionRefusesTheDefaultSenderWhenUsersGetEmail(t *testing.T) {
	t.Parallel()

	for _, cfg := range []config.Choices{{Stack: config.StackStandard}, {AddOns: config.NewSet(config.AllAuth)}} {
		if guard := (&Env{cfg: mustSpec(cfg)}).fromEmailGuard(""); !strings.Contains(guard, `if DEFAULT_FROM_EMAIL == "`+djangoDefaultFromEmail+`":`) {
			t.Errorf("%+v: production would send account email from %s:\n%s", cfg, djangoDefaultFromEmail, guard)
		}
	}
	if guard := (&Env{}).fromEmailGuard(""); guard != "" {
		t.Errorf("an API-only project sends no account email, so it needs no sender:\n%s", guard)
	}
}

func TestBooleanSettingsRejectWhatTheyDoNotRecognise(t *testing.T) {
	t.Parallel()

	settings := settingsWithObservability(t, observabilityConfig())
	if strings.Contains(settings, "env.bool(") {
		t.Errorf("django-environ's env.bool turns a typo into False; booleans must go through env_bool:\n%s", settings)
	}
	if !strings.Contains(settings, "raise ImproperlyConfigured(message)") || !strings.Contains(settings, "def env_bool(") {
		t.Errorf("env_bool is missing or does not refuse an unknown value:\n%s", settings)
	}
}
