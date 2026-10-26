package scaffold

import (
	"github.com/vaibdix/django-stack/internal/config"
)

func (p project) sentrySettingsBlock() string {
	r := p.env()
	return render("sentry.py.tmpl", struct{ DSN, Environment, Release, SendPII, Traces, Profiles string }{
		r.secret("SENTRY_DSN"),
		r.sentryEnvironment(),
		r.str("SENTRY_RELEASE", ""),
		r.boolean("SENTRY_SEND_PII", false),
		r.number("SENTRY_TRACES_SAMPLE_RATE", 0.0),
		r.number("SENTRY_PROFILE_SESSION_SAMPLE_RATE", 0.0),
	})
}

func sentryExampleSection(environment string) string {
	return render("env_sentry.tmpl", struct{ Environment string }{environment})
}

func (p project) deployedSentrySection(environment string) string {
	if !p.cfg.Has(config.Sentry) {
		return ""
	}
	return "\n" + sentryExampleSection(environment) + "\n"
}

func (r envReader) sentryEnvironment() string {
	if r.pydantic {
		return "env.SENTRY_ENVIRONMENT or ENVIRONMENT"
	}
	return `env("SENTRY_ENVIRONMENT", default=ENVIRONMENT)`
}

func sentryFeature() feature {
	return feature{
		enabled: has(config.Sentry),
		contribute: func(p project) contribution {
			return contribution{
				runtime:  []string{pkgSentry},
				readme:   func(d *readmeData) { d.Sentry = true },
				settings: settingsContribution{imports: []string{"import sentry_sdk"}, blocks: []string{p.sentrySettingsBlock()}},
			}
		},
	}
}
