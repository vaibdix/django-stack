package scaffold

import "fmt"

func (e *Env) applySentry(projectPath string) error {
	if !e.Cfg.SetupSentry {
		return nil
	}

	se, err := e.settingsEditor(projectPath)
	if err != nil {
		return err
	}

	if err := se.AddImport("import sentry_sdk"); err != nil {
		return err
	}
	se.AppendBlock(e.sentrySettingsBlock())

	if err := se.Save(); err != nil {
		return err
	}

	e.Step("Sentry configured. It stays inert until SENTRY_DSN is set in .env.")
	return nil
}

func (e *Env) sentrySettingsBlock() string {
	r := e.env()

	return fmt.Sprintf(`
SENTRY_DSN = %s
SENTRY_ENVIRONMENT = %s
SENTRY_RELEASE = %s
SENTRY_SEND_PII = %s
SENTRY_TRACES_SAMPLE_RATE = %s
SENTRY_PROFILE_SESSION_SAMPLE_RATE = %s

if SENTRY_DSN:
    sentry_sdk.init(
        dsn=SENTRY_DSN,
        environment=SENTRY_ENVIRONMENT,
        release=SENTRY_RELEASE or None,
        send_default_pii=SENTRY_SEND_PII,
        traces_sample_rate=SENTRY_TRACES_SAMPLE_RATE,
        profile_session_sample_rate=SENTRY_PROFILE_SESSION_SAMPLE_RATE,
        profile_lifecycle='trace',
        enable_logs=True,
    )
`,
		r.secret("SENTRY_DSN"),
		r.sentryEnvironment(),
		r.str("SENTRY_RELEASE", ""),
		r.boolean("SENTRY_SEND_PII", false),
		r.number("SENTRY_TRACES_SAMPLE_RATE", 0.0),
		r.number("SENTRY_PROFILE_SESSION_SAMPLE_RATE", 0.0),
	)
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
	if !e.Cfg.SetupSentry {
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
