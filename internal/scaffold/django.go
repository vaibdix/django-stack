package scaffold

import (
	"context"
	"fmt"
	"runtime"
	"strings"

	"github.com/vaibdix/django-stack/internal/config"
	"github.com/vaibdix/django-stack/internal/pyenv"
)

func (e *Env) createDjangoProject(ctx context.Context) error {
	if err := e.emit(e.projectFiles()); err != nil {
		return err
	}
	e.Step(fmt.Sprintf("Django project '%s' created.", e.cfg.Name))
	return nil
}

func (e *Env) reportPerformanceTip() {
	if e.toolchain == pyenv.Pip && pyenv.IsWindows(runtime.GOOS) {
		e.Step(
			"Tip: Install 'uv' for faster package management: pip install uv")
	}
}

func consoleLoggingBlock() string { return renderStatic("console_logging.py.tmpl") }

const manifestStaticStorage = "whitenoise.storage.CompressedManifestStaticFilesStorage"

const djangoDefaultFromEmail = "webmaster@localhost"

func (e *Env) sendsAccountEmail() bool {
	return e.cfg.Has(config.AllAuth) || e.cfg.CreateTemplates()
}

func (e *Env) fromEmailGuard(indent string) string {
	if !e.sendsAccountEmail() {
		return ""
	}
	return indent + "if DEFAULT_FROM_EMAIL == '" + djangoDefaultFromEmail + "':\n" +
		indent + "    message = 'DEFAULT_FROM_EMAIL is still " + djangoDefaultFromEmail + "; set it to an address your mail server sends from.'\n" +
		indent + "    raise ImproperlyConfigured(message)\n"
}

func separated(block string) string {
	if block == "" {
		return ""
	}
	return "\n" + block
}

func (e *Env) singleFileEnvironmentBlock() string {
	env := e.productionEnvExpressions()

	var development []string
	if e.cfg.CreateTemplates() {
		development = append(development,
			"    INSTALLED_APPS = [*INSTALLED_APPS, '"+appBrowserReload+"']",
			"    MIDDLEWARE = [*MIDDLEWARE, '"+middlewareBrowserReload+"']")
	}

	console := "    EMAIL_BACKEND = 'django.core.mail.backends.console.EmailBackend'"
	smtp := fmt.Sprintf(`    EMAIL_BACKEND = 'django.core.mail.backends.smtp.EmailBackend'
    EMAIL_HOST = %s
    EMAIL_PORT = %s
    EMAIL_HOST_USER = %s
    EMAIL_HOST_PASSWORD = %s
    EMAIL_USE_TLS = %s`, env.EmailHost, env.EmailPort, env.EmailUser, env.EmailPassword, env.EmailUseTLS)
	if config.DjangoSupportsMailers(e.cfg.Django) {
		console = `    MAILERS: dict[str, dict[str, object]] = {
        'default': {
            'BACKEND': 'django.core.mail.backends.console.EmailBackend',
        },
    }`
		smtp = fmt.Sprintf(`    MAILERS = {
        'default': {
            'BACKEND': 'django.core.mail.backends.smtp.EmailBackend',
            'OPTIONS': {
                'host': %s,
                'port': %s,
                'username': %s,
                'password': %s,
                'use_tls': %s,
            },
        },
    }`, env.EmailHost, env.EmailPort, env.EmailUser, env.EmailPassword, env.EmailUseTLS)
	}
	development = append(development, console)

	return `
DEFAULT_FROM_EMAIL = ` + env.DefaultFromEmail + `

if DEBUG:
` + strings.Join(development, "\n") + `
else:
    if SECRET_KEY == '` + placeholderSecretKey + `':` + e.noqa("S105") + `
        message = 'SECRET_KEY is still the .env.example placeholder; generate a real one before deploying.'
        raise ImproperlyConfigured(message)
` + separated(e.fromEmailGuard("    ")) + `
    SECURE_SSL_REDIRECT = ` + env.SSLRedirect + `
    SECURE_REDIRECT_EXEMPT = [r'^healthz/$']
    SECURE_HSTS_SECONDS = ` + env.HSTSSeconds + `
    SECURE_HSTS_INCLUDE_SUBDOMAINS = ` + env.HSTSSubdomains + `
    SECURE_HSTS_PRELOAD = ` + env.HSTSPreload + `
    if ` + env.TrustProxyHeader + `:
        SECURE_PROXY_SSL_HEADER = ('HTTP_X_FORWARDED_PROTO', 'https')
    SESSION_COOKIE_SECURE = True
    CSRF_COOKIE_SECURE = True
    CSRF_TRUSTED_ORIGINS = ` + env.CSRFOrigins + `
` + enforceCSP(config.DjangoSupportsNativeCSP(e.cfg.Django), "    ") + `

` + smtp + `

    CACHES = {
        'default': {
            'BACKEND': 'django.core.cache.backends.db.DatabaseCache',
            'LOCATION': 'django_cache_table',
        },
    }
    STORAGES = {
        'default': {
            'BACKEND': 'django.core.files.storage.FileSystemStorage',
        },
        'staticfiles': {
            'BACKEND': '` + manifestStaticStorage + `',
        },
    }
`
}

func (e *Env) noqa(codes string) string {
	if !e.cfg.Has(config.Ruff) {
		return ""
	}
	return "  # noqa: " + codes
}
