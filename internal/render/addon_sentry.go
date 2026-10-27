package render

import (
	"github.com/vaibdix/django-stack/internal/config"
)

func (p Project) sentrySettingsBlock() string {
	r := p.env()
	return Render("sentry.py.tmpl", struct{ DSN, Environment, Release, SendPII, Traces, Profiles string }{
		r.secret("SENTRY_DSN"),
		r.sentryEnvironment(),
		r.str("SENTRY_RELEASE", ""),
		r.boolean("SENTRY_SEND_PII", false),
		r.number("SENTRY_TRACES_SAMPLE_RATE", 0.0),
		r.number("SENTRY_PROFILE_SESSION_SAMPLE_RATE", 0.0),
	})
}

func sentryExampleSection(environment string) string {
	return Render("env_sentry.tmpl", struct{ Environment string }{environment})
}

func (p Project) deployedSentrySection(environment string) string {
	if !p.Cfg.Has(config.Sentry) {
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
		enabled: Has(config.Sentry),
		contribute: func(p Project) contribution {
			return contribution{
				runtime:  []string{PkgSentry},
				readme:   func(d *readmeData) { d.Sentry = true },
				settings: settingsContribution{imports: []string{"import sentry_sdk"}, blocks: []block{{rank: rankSentry, text: p.sentrySettingsBlock()}}},
			}
		},
	}
}
