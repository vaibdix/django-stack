package scaffold

import (
	"strings"
	"testing"

	"github.com/vaibdix/django-stack/internal/config"
)

func TestSentryReportsUnderTheProjectEnvironmentByDefault(t *testing.T) {
	t.Parallel()

	for _, pydantic := range []bool{false, true} {
		block := (&Env{cfg: config.Project{SetupSentry: true, SetupPydanticConf: pydantic}}).sentrySettingsBlock()
		if !strings.Contains(block, "ENVIRONMENT)") && !strings.Contains(block, "or ENVIRONMENT") {
			t.Errorf("pydantic=%v: SENTRY_ENVIRONMENT does not fall back to ENVIRONMENT, so production "+
				"events are filed under a hard-coded name:\n%s", pydantic, block)
		}
		if strings.Contains(block, "'local'") {
			t.Errorf("pydantic=%v: SENTRY_ENVIRONMENT still defaults to 'local':\n%s", pydantic, block)
		}
	}
}
