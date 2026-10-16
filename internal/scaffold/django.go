package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/vaibdix/django-cli/internal/config"
	"github.com/vaibdix/django-cli/internal/scaffold/pyenv"
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

	if err := e.stripCommentsFromDjangoBoilerplate(projectPath); err != nil {
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

	envCfg := e.envConfig()
	se.InjectEnvConfig(envCfg)
	if err := se.InjectSecuritySettings(config.DjangoSupportsNativeCSP(e.Cfg.DjangoVersion), envCfg); err != nil {
		return err
	}

	if err := se.AddToListAfter(MiddlewareList, MiddlewareSecurity, MiddlewareWhiteNoise); err != nil {
		return fmt.Errorf("failed to add %s to %s: %w", MiddlewareWhiteNoise, MiddlewareList, err)
	}

	if e.Cfg.CreateTemplates && !e.Cfg.SetupMultiEnv && !se.Contains(MiddlewareBrowserReload) {
		se.AppendBlock(browserReloadBlock)
	}
	if !e.Cfg.SetupMultiEnv && !se.Contains("DEFAULT_FROM_EMAIL") {
		se.content = stripSettingBlock(se.content, "MAILERS")
		se.AppendBlock(e.singleFileMailBlock())
	}
	if !e.Cfg.SetupMultiEnv && !se.Contains(manifestStaticStorage) {
		se.AppendBlock(singleFileProductionBlock)
	}
	if !e.Cfg.SetupMultiEnv && !se.Contains(placeholderSecretKey) {
		se.AddImport("from django.core.exceptions import ImproperlyConfigured")
		se.AppendBlock(singleFilePlaceholderGuard)
	}

	if !e.Cfg.CreateTemplates && !se.Contains("STATIC_ROOT") {
		// API-only projects still serve the admin's static files, and
		// collectstatic refuses to run without somewhere to put them.
		se.content = strings.Replace(se.content, "STATIC_URL = 'static/'",
			"STATIC_URL = 'static/'\n\nSTATIC_ROOT = BASE_DIR / 'staticfiles'", 1)
		if !se.Contains("STATIC_ROOT") {
			se.SetTopLevel("STATIC_ROOT", "BASE_DIR / 'staticfiles'")
		}
	}

	if e.Cfg.CreateTemplates && !e.Cfg.SetupAllAuth {
		se.SetTopLevel("LOGIN_URL", "'login'")
		se.SetTopLevel("LOGIN_REDIRECT_URL", "'home'")
		se.SetTopLevel("LOGOUT_REDIRECT_URL", "'home'")
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
	if !pyenv.UVAvailable() && runtime.GOOS == "windows" {
		e.Step(
			"Tip: Install 'uv' for faster package management: pip install uv")
	}
}

// consoleLoggingBlock is the console example from Django's logging topic.
// Without it, Django's default LOGGING prints to the console only while DEBUG
// is on and sends production errors to mail_admins, which reaches nobody when
// ADMINS is empty — so a 500 would leave no traceback anywhere.
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

// singleFileProductionBlock mirrors settings/production.py: hashed,
// compressed static files that WhiteNoise can cache forever, and a cache
// shared by every worker so throttles and rate limits add up.
const singleFileProductionBlock = `
if not DEBUG:
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

// singleFilePlaceholderGuard stops a deploy that copied .env.example and
// never replaced its SECRET_KEY, which passes every length check.
const singleFilePlaceholderGuard = `
if not DEBUG and SECRET_KEY == '` + placeholderSecretKey + `':  # noqa: S105
    message = 'SECRET_KEY is still the .env.example placeholder; generate a real one before deploying.'
    raise ImproperlyConfigured(message)
`

const browserReloadBlock = `
if DEBUG:
    INSTALLED_APPS = [*INSTALLED_APPS, '` + AppBrowserReload + `']
    MIDDLEWARE = [*MIDDLEWARE, '` + MiddlewareBrowserReload + `']
`

func (e *Env) singleFileMailBlock() string {
	env := e.productionEnvExpressions()
	smtp := fmt.Sprintf(`    EMAIL_BACKEND = 'django.core.mail.backends.smtp.EmailBackend'
    EMAIL_HOST = %s
    EMAIL_PORT = %s
    EMAIL_HOST_USER = %s
    EMAIL_HOST_PASSWORD = %s
    EMAIL_USE_TLS = %s`, env.EmailHost, env.EmailPort, env.EmailUser, env.EmailPassword, env.EmailUseTLS)
	console := "    EMAIL_BACKEND = 'django.core.mail.backends.console.EmailBackend'"

	if config.DjangoSupportsMailers(e.Cfg.DjangoVersion) {
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
		console = `    MAILERS: dict[str, dict[str, object]] = {
        'default': {
            'BACKEND': 'django.core.mail.backends.console.EmailBackend',
        },
    }`
	}

	return `
DEFAULT_FROM_EMAIL = ` + env.DefaultFromEmail + `

if DEBUG:
` + console + `
else:
` + smtp + `
`
}
