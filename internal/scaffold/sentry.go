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

	se.AddImport("import sentry_sdk")
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
# =============================================================================
# Error Tracking (Sentry)
# =============================================================================
# DjangoIntegration is auto-enabled by the SDK — adding it explicitly is only
# for overriding its options, so this init deliberately does not list it.
SENTRY_DSN = %s

if SENTRY_DSN:
    sentry_sdk.init(
        dsn=SENTRY_DSN,
        environment=%s,
        release=%s or None,
        # Off by default: send_default_pii ships user email and IP address to a
        # third party, which is not a choice this scaffold should make for you.
        send_default_pii=%s,
        traces_sample_rate=%s,
        profile_session_sample_rate=%s,
        profile_lifecycle='trace',
        enable_logs=True,
    )
`,
		r.str("SENTRY_DSN", ""),
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

// deployedSentrySection is for the .env.production/.staging examples, which
// dotenvSection does not assemble — so it carries its own separator and
// trailing newline, and names the environment the file is for rather than
// copying the local default into a deployed one.
func (e *Env) deployedSentrySection(environment string) string {
	if !e.Cfg.SetupSentry {
		return ""
	}
	return "\n" + sentryExampleSection(environment) + "\n"
}
