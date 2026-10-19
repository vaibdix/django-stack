package scaffold

const (
	TemplatesDir = "templates"
)

const (
	SettingsFile        = "settings.py"
	ManageFile          = "manage.py"
	URLsFile            = "urls.py"
	EnvFile             = ".env"
	EnvExampleFile      = ".env.example"
	PyprojectFile       = "pyproject.toml"
	PreCommitConfigFile = ".pre-commit-config.yaml"
	GitignoreFile       = ".gitignore"
)

const (
	InstalledApps  = "INSTALLED_APPS"
	MiddlewareList = "MIDDLEWARE"
)

const (
	PkgRestFramework  = "djangorestframework"
	PkgNinja          = "django-ninja"
	PkgAllAuth        = "django-allauth"
	PkgHTMX           = "django-htmx"
	PkgBrowserReload  = "django-browser-reload"
	PkgEnviron        = "django-environ"
	PkgPsycopg        = "psycopg[binary]"
	PkgMySQLClient    = "mysqlclient"
	PkgArgon2         = "argon2-cffi"
	PkgRuff           = "ruff"
	PkgDjLint         = "djlint"
	PkgMypy           = "mypy"
	PkgDjangoStubs    = "django-stubs"
	PkgDjangoStubsExt = "django-stubs-ext"
	PkgDRFStubs       = "djangorestframework-stubs"
	PkgPreCommit      = "pre-commit"
	PkgDRFSpectacular = "drf-spectacular"

	PkgDRFSpectacularSidecar = "drf-spectacular-sidecar"

	PkgPydantic         = "pydantic"
	PkgPydanticSettings = "pydantic-settings"
	PkgStructlog        = "structlog"
	PkgDjangoStructlog  = "django-structlog"
	PkgSentry           = "sentry-sdk"
	PkgDjangoCSP        = "django-csp"
	PkgWhiteNoise       = "whitenoise"
	PkgGunicorn         = "gunicorn"
)

const (
	AppRestFramework  = "rest_framework"
	AppAllAuth        = "allauth"
	AppAllAuthAccount = "allauth.account"
	AppHTMX           = "django_htmx"
	AppBrowserReload  = "django_browser_reload"
	AppDRFSpectacular = "drf_spectacular"

	AppDRFSpectacularSidecar = "drf_spectacular_sidecar"

	AppNinja     = "ninja"
	AppDjangoCSP = "csp"
	AppStructlog = "django_structlog"
)

const (
	MiddlewareAllAuth       = "allauth.account.middleware.AccountMiddleware"
	MiddlewareHTMX          = "django_htmx.middleware.HtmxMiddleware"
	MiddlewareBrowserReload = "django_browser_reload.middleware.BrowserReloadMiddleware"
	MiddlewareCSP           = "django.middleware.csp.ContentSecurityPolicyMiddleware"
	MiddlewareDjangoCSP     = "csp.middleware.CSPMiddleware"
	MiddlewareStructlog     = "django_structlog.middlewares.RequestMiddleware"
	MiddlewareWhiteNoise    = "whitenoise.middleware.WhiteNoiseMiddleware"
	MiddlewareSecurity      = "django.middleware.security.SecurityMiddleware"
)

const ConfFile = "conf.py"

const DjLintVersion = "1.46.6"
