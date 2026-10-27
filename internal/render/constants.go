package render

const (
	templatesDir = "templates"
)

const (
	SettingsFile        = "settings.py"
	ManageFile          = "manage.py"
	urlsFile            = "urls.py"
	EnvFile             = ".env"
	EnvExampleFile      = ".env.example"
	PyprojectFile       = "pyproject.toml"
	PreCommitConfigFile = ".pre-commit-config.yaml"
	GitignoreFile       = ".gitignore"
	PackageJSONFile     = "package.json"
)

const (
	PkgRestFramework  = "djangorestframework"
	pkgNinja          = "django-ninja"
	PkgAllAuth        = "django-allauth"
	PkgHTMX           = "django-htmx"
	pkgBrowserReload  = "django-browser-reload"
	PkgEnviron        = "django-environ"
	pkgArgon2         = "argon2-cffi"
	PkgRuff           = "ruff"
	PkgDjLint         = "djlint"
	PkgMypy           = "mypy"
	pkgDjangoStubs    = "django-stubs"
	PkgDjangoStubsExt = "django-stubs-ext"
	PkgDRFStubs       = "djangorestframework-stubs"
	PkgPreCommit      = "pre-commit"
	pkgDRFSpectacular = "drf-spectacular"

	pkgDRFSpectacularSidecar = "drf-spectacular-sidecar"

	pkgPydantic         = "pydantic"
	pkgPydanticSettings = "pydantic-settings"
	PkgStructlog        = "structlog"
	pkgDjangoStructlog  = "django-structlog"
	PkgSentry           = "sentry-sdk"
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

const ConfFile = "conf.py"
