package scaffold

const (
	templatesDir = "templates"
)

const (
	settingsFile        = "settings.py"
	manageFile          = "manage.py"
	urlsFile            = "urls.py"
	envFile             = ".env"
	envExampleFile      = ".env.example"
	pyprojectFile       = "pyproject.toml"
	preCommitConfigFile = ".pre-commit-config.yaml"
	gitignoreFile       = ".gitignore"
	packageJSONFile     = "package.json"
)

const (
	pkgRestFramework  = "djangorestframework"
	pkgNinja          = "django-ninja"
	pkgAllAuth        = "django-allauth"
	pkgHTMX           = "django-htmx"
	pkgBrowserReload  = "django-browser-reload"
	pkgEnviron        = "django-environ"
	pkgArgon2         = "argon2-cffi"
	pkgRuff           = "ruff"
	pkgDjLint         = "djlint"
	pkgMypy           = "mypy"
	pkgDjangoStubs    = "django-stubs"
	pkgDjangoStubsExt = "django-stubs-ext"
	pkgDRFStubs       = "djangorestframework-stubs"
	pkgPreCommit      = "pre-commit"
	pkgDRFSpectacular = "drf-spectacular"

	pkgDRFSpectacularSidecar = "drf-spectacular-sidecar"

	pkgPydantic         = "pydantic"
	pkgPydanticSettings = "pydantic-settings"
	pkgStructlog        = "structlog"
	pkgDjangoStructlog  = "django-structlog"
	pkgSentry           = "sentry-sdk"
	pkgDjangoCSP        = "django-csp"
	pkgWhiteNoise       = "whitenoise"
	pkgGunicorn         = "gunicorn"
)

const (
	appRestFramework  = "rest_framework"
	appAllAuth        = "allauth"
	appAllAuthAccount = "allauth.account"
	appHTMX           = "django_htmx"
	appBrowserReload  = "django_browser_reload"
	appDRFSpectacular = "drf_spectacular"

	appDRFSpectacularSidecar = "drf_spectacular_sidecar"

	appNinja     = "ninja"
	appDjangoCSP = "csp"
	appStructlog = "django_structlog"
)

const (
	middlewareAllAuth       = "allauth.account.middleware.AccountMiddleware"
	middlewareHTMX          = "django_htmx.middleware.HtmxMiddleware"
	middlewareBrowserReload = "django_browser_reload.middleware.BrowserReloadMiddleware"
	middlewareCSP           = "django.middleware.csp.ContentSecurityPolicyMiddleware"
	middlewareDjangoCSP     = "csp.middleware.CSPMiddleware"
	middlewareStructlog     = "django_structlog.middlewares.RequestMiddleware"
	middlewareWhiteNoise    = "whitenoise.middleware.WhiteNoiseMiddleware"
	middlewareSecurity      = "django.middleware.security.SecurityMiddleware"
)

const confFile = "conf.py"

const djLintVersion = "1.47.0"
