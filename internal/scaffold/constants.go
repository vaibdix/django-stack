package scaffold

const (
	TemplatesDir = "templates"
	StaticDir    = "static"
)

const (
	SettingsFile        = "settings.py"
	ManageFile          = "manage.py"
	URLsFile            = "urls.py"
	EnvFile             = ".env"
	EnvExampleFile      = ".env.example"
	PyprojectFile       = "pyproject.toml"
	PreCommitConfigFile = ".pre-commit-config.yaml"
)

const (
	InstalledApps  = "INSTALLED_APPS"
	MiddlewareList = "MIDDLEWARE"
)

const (
	PkgRestFramework  = "djangorestframework"
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
	PkgDRFStubs       = "djangorestframework-stubs"
	PkgPreCommit      = "pre-commit"
	PkgDRFSpectacular = "drf-spectacular"
)

const (
	AppRestFramework  = "rest_framework"
	AppAllAuth        = "allauth"
	AppAllAuthAccount = "allauth.account"
	AppContribSites   = "django.contrib.sites"
	AppHTMX           = "django_htmx"
	AppBrowserReload  = "django_browser_reload"
	AppDRFSpectacular = "drf_spectacular"
)

const (
	MiddlewareAllAuth       = "allauth.account.middleware.AccountMiddleware"
	MiddlewareHTMX          = "django_htmx.middleware.HtmxMiddleware"
	MiddlewareBrowserReload = "django_browser_reload.middleware.BrowserReloadMiddleware"
	MiddlewareCSP           = "django.middleware.csp.ContentSecurityPolicyMiddleware"
)
