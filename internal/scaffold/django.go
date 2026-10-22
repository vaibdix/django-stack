package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/vaibdix/django-stack/internal/config"
	"github.com/vaibdix/django-stack/internal/pyenv"
)

func (e *Env) createDjangoProject(projectPath string) error {
	ctx, cancel := e.timeoutContext()
	defer cancel()

	pythonVenvPath := pyenv.InterpreterPath(projectPath)
	if _, err := e.runIn(ctx, projectPath, pythonVenvPath, "-m", "django", "startproject", e.Cfg.ModuleName(), "."); err != nil {
		return fmt.Errorf("failed to create Django project: %w", err)
	}

	if err := annotateManageMain(projectPath); err != nil {
		return err
	}

	e.Step(fmt.Sprintf("Django project '%s' created.", e.Cfg.ProjectName))
	return nil
}

func annotateManageMain(projectPath string) error {
	path := filepath.Join(projectPath, ManageFile)
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", ManageFile, err)
	}

	content := strings.Replace(string(data), "def main():", "def main() -> None:", 1)
	if err := os.WriteFile(path, []byte(content), 0755); err != nil {
		return fmt.Errorf("failed to write %s: %w", ManageFile, err)
	}
	return nil
}

func (e *Env) configureSettings(settingsPath string) error {
	se, err := NewSettingsEditor(settingsPath)
	if err != nil {
		return err
	}

	if err := se.InjectEnvConfig(e.envConfig()); err != nil {
		return err
	}
	if err := se.InjectSecuritySettings(config.DjangoSupportsNativeCSP(e.Cfg.DjangoVersion)); err != nil {
		return err
	}

	if err := se.AddToListAfter(MiddlewareList, MiddlewareSecurity, MiddlewareWhiteNoise); err != nil {
		return fmt.Errorf("failed to add %s to %s: %w", MiddlewareWhiteNoise, MiddlewareList, err)
	}

	if !e.Cfg.SetupMultiEnv && !se.Contains("DEFAULT_FROM_EMAIL") {
		se.StripSettingBlock("MAILERS")
		if err := se.AddImport("from django.core.exceptions import ImproperlyConfigured"); err != nil {
			return err
		}
		se.AppendBlock(e.singleFileEnvironmentBlock())
	}

	if !e.Cfg.CreateTemplates && !se.Contains("STATIC_ROOT") {
		if err := se.InsertAfterLine(staticURLPrefix, "STATIC_ROOT = BASE_DIR / 'staticfiles'"); err != nil {
			return err
		}
	}

	if e.Cfg.CreateTemplates && !e.Cfg.SetupAllAuth {
		se.SetTopLevel("LOGIN_URL", "'login'")
		if err := se.SetTopLevelAfter("LOGIN_URL", "LOGIN_REDIRECT_URL", "'home'"); err != nil {
			return err
		}
		if err := se.SetTopLevelAfter("LOGIN_REDIRECT_URL", "LOGOUT_REDIRECT_URL", "'home'"); err != nil {
			return err
		}
	}

	if !e.Cfg.SetupStructlog && !se.Contains("LOGGING = ") {
		se.AppendBlock(consoleLoggingBlock)
	}

	if e.Cfg.CreateTemplates {
		if err := se.AddProjectContextProcessor(e.Cfg.ModuleName()); err != nil {
			return err
		}
	}

	if err := se.Save(); err != nil {
		return err
	}
	e.Step("Django settings configured.")
	if e.Cfg.SetupPydanticConf {
		e.Step("Environment variables configured via pydantic-settings.")
	} else {
		e.Step("Environment variables configured via django-environ.")
	}
	return nil
}

func (e *Env) reportPerformanceTip() {
	if e.toolchain == pyenv.Pip && runtime.GOOS == "windows" {
		e.Step(
			"Tip: Install 'uv' for faster package management: pip install uv")
	}
}

const consoleLoggingBlock = `
LOGGING = {
    'version': 1,
    'disable_existing_loggers': False,
    'handlers': {
        'console': {
            'class': 'logging.StreamHandler',
        },
    },
    'root': {
        'handlers': ['console'],
        'level': 'WARNING',
    },
    'loggers': {
        'django': {
            'handlers': ['console'],
            'level': 'INFO',
            'propagate': False,
        },
    },
}
`

const manifestStaticStorage = "whitenoise.storage.CompressedManifestStaticFilesStorage"

const djangoDefaultFromEmail = "webmaster@localhost"

func (e *Env) sendsAccountEmail() bool {
	return e.Cfg.SetupAllAuth || e.Cfg.CreateTemplates
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
	if e.Cfg.CreateTemplates {
		development = append(development,
			"    INSTALLED_APPS = [*INSTALLED_APPS, '"+AppBrowserReload+"']",
			"    MIDDLEWARE = [*MIDDLEWARE, '"+MiddlewareBrowserReload+"']")
	}

	console := "    EMAIL_BACKEND = 'django.core.mail.backends.console.EmailBackend'"
	smtp := fmt.Sprintf(`    EMAIL_BACKEND = 'django.core.mail.backends.smtp.EmailBackend'
    EMAIL_HOST = %s
    EMAIL_PORT = %s
    EMAIL_HOST_USER = %s
    EMAIL_HOST_PASSWORD = %s
    EMAIL_USE_TLS = %s`, env.EmailHost, env.EmailPort, env.EmailUser, env.EmailPassword, env.EmailUseTLS)
	if config.DjangoSupportsMailers(e.Cfg.DjangoVersion) {
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
` + enforceCSP(config.DjangoSupportsNativeCSP(e.Cfg.DjangoVersion), "    ") + `

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
	if !e.Cfg.SetupRuff {
		return ""
	}
	return "  # noqa: " + codes
}
