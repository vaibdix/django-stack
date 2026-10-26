package scaffold

import (
	"github.com/vaibdix/django-stack/internal/config"
)

func (e *Env) sentrySettingsBlock() string {
	r := e.env()
	return render("sentry.py.tmpl", struct{ DSN, Environment, Release, SendPII, Traces, Profiles string }{
		r.secret("SENTRY_DSN"),
		r.sentryEnvironment(),
		r.str("SENTRY_RELEASE", ""),
		r.boolean("SENTRY_SEND_PII", false),
		r.number("SENTRY_TRACES_SAMPLE_RATE", 0.0),
		r.number("SENTRY_PROFILE_SESSION_SAMPLE_RATE", 0.0),
	})
}

const sentrySectionHeading = "# === Error Tracking (Sentry - uncomment when you have a project) ==="

func sentryExampleSection(environment string) string {
	return sentrySectionHeading + `
# SENTRY_DSN=
# SENTRY_ENVIRONMENT defaults to ENVIRONMENT; set it only to report under another name.
# SENTRY_ENVIRONMENT=` + environment + `
# SENTRY_RELEASE=
# SENTRY_SEND_PII=False
# SENTRY_TRACES_SAMPLE_RATE=0.1
# SENTRY_PROFILE_SESSION_SAMPLE_RATE=0.1`
}

func (e *Env) deployedSentrySection(environment string) string {
	if !e.cfg.Has(config.Sentry) {
		return ""
	}
	return "\n" + sentryExampleSection(environment) + "\n"
}

func (r envReader) sentryEnvironment() string {
	if r.pydantic {
		return "env.SENTRY_ENVIRONMENT or ENVIRONMENT"
	}
	return "env('SENTRY_ENVIRONMENT', default=ENVIRONMENT)"
}

func sentryAddOn() addOn {
	return addOn{
		enabled: has(config.Sentry),
		contribute: func(e *Env) contribution {
			return contribution{
				runtime:  []string{pkgSentry},
				readme:   func(d *readmeData) { d.Sentry = true },
				settings: settingsContribution{imports: []string{"import sentry_sdk"}, blocks: []string{e.sentrySettingsBlock()}},
			}
		},
	}
}
