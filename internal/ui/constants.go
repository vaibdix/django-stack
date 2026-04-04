package ui

// Directory names used during project scaffolding.
const (
	VenvDir      = ".venv"
	TemplatesDir = "templates"
	StaticDir    = "static"
)

// File names used during project scaffolding.
const (
	SettingsFile        = "settings.py"
	ManageFile          = "manage.py"
	URLsFile            = "urls.py"
	EnvFile             = ".env"
	EnvExampleFile      = ".env.example"
	PyprojectFile       = "pyproject.toml"
	PreCommitConfigFile = ".pre-commit-config.yaml"
)

// Django settings list names.
const (
	InstalledApps  = "INSTALLED_APPS"
	MiddlewareList = "MIDDLEWARE"
)

// Pip package names.
const (
	PkgRestFramework = "djangorestframework"
	PkgAllAuth       = "django-allauth"
	PkgHTMX          = "django-htmx"
	PkgBrowserReload = "django-browser-reload"
	PkgEnviron       = "django-environ"
	PkgDotenv        = "python-dotenv"
	PkgPsycopg       = "psycopg2-binary"
	PkgMySQLClient   = "mysqlclient"
	PkgArgon2        = "argon2-cffi"
	PkgRuff          = "ruff"
	PkgDjLint        = "djlint"
	PkgMypy          = "mypy"
	PkgDjangoStubs   = "django-stubs"
	PkgDRFStubs      = "djangorestframework-stubs"
	PkgPreCommit     = "pre-commit"
	PkgDRFSpectacular = "drf-spectacular"
)

// Django app names for INSTALLED_APPS.
const (
	AppRestFramework = "rest_framework"
	AppAllAuth       = "allauth"
	AppAllAuthAccount = "allauth.account"
	AppContribSites  = "django.contrib.sites"
	AppHTMX          = "django_htmx"
	AppBrowserReload = "django_browser_reload"
	AppDRFSpectacular = "drf_spectacular"
)

// Middleware class paths.
const (
	MiddlewareAllAuth        = "allauth.account.middleware.AccountMiddleware"
	MiddlewareHTMX           = "django_htmx.middleware.HtmxMiddleware"
	MiddlewareBrowserReload  = "django_browser_reload.middleware.BrowserReloadMiddleware"
)
