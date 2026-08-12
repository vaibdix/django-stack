package scaffold

import "fmt"

func (e *Env) applySentry(projectPath string) error {
	if !e.Cfg.SetupSentry {
		return nil
	}

	if err := e.installPackages(projectPath, "sentry-sdk", PkgSentry); err != nil {
		return err
	}

	se, err := e.settingsEditor(projectPath)
	if err != nil {
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
`,
		r.secret("SENTRY_DSN", ""),
		r.str("SENTRY_ENVIRONMENT", "local"),
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
# SENTRY_ENVIRONMENT=` + environment + `
# SENTRY_RELEASE=
# SENTRY_SEND_PII=False
# SENTRY_TRACES_SAMPLE_RATE=0.1
# SENTRY_PROFILE_SESSION_SAMPLE_RATE=0.1`
}

func (e *Env) dotenvSentrySection() string {
	if !e.Cfg.SetupSentry {
		return ""
	}
	return sentryExampleSection("local")
}

func (e *Env) deployedSentrySection(environment string) string {
	if !e.Cfg.SetupSentry {
		return ""
	}
	return "\n" + sentryExampleSection(environment) + "\n"
}
