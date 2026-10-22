package config

import "fmt"

var installedModules = map[string]string{
	"django":                "Django",
	"asgiref":               "asgiref",
	"sqlparse":              "sqlparse",
	"django_htmx":           "django-htmx",
	"django_browser_reload": "django-browser-reload",
	"environ":               "django-environ",
	"rest_framework":        "djangorestframework",
	"drf_spectacular":       "drf-spectacular",
	"ninja":                 "django-ninja",
	"allauth":               "django-allauth",
	"argon2":                "argon2-cffi",
	"psycopg":               "psycopg",
	"MySQLdb":               "mysqlclient",
	"whitenoise":            "whitenoise",
	"gunicorn":              "gunicorn",
	"pydantic":              "pydantic",
	"pydantic_core":         "pydantic-core",
	"pydantic_settings":     "pydantic-settings",
	"structlog":             "structlog",
	"django_structlog":      "django-structlog",
	"ipware":                "django-ipware",
	"sentry_sdk":            "sentry-sdk",
	"django_stubs_ext":      "django-stubs-ext",
	"mypy_django_plugin":    "django-stubs",
	"mypy_drf_plugin":       "djangorestframework-stubs",
	"pre_commit":            "pre-commit",
	"typing_extensions":     "typing-extensions",
	"ruff":                  "ruff",
	"djlint":                "djlint",
	"mypy":                  "mypy",
}

var scaffoldOwnedNames = map[string]string{
	"accounts":    "the custom user app",
	"core":        "the shared base-model app",
	"templates":   "the global templates directory",
	"static":      "the static files directory",
	"staticfiles": "the collectstatic target",
}

func shadowsInstalledModule(moduleName string) (string, bool) {
	distribution, shadowed := installedModules[moduleName]
	return distribution, shadowed
}

func validateNoModuleShadowing(name, moduleName string) error {
	if owner, taken := scaffoldOwnedNames[moduleName]; taken {
		return fmt.Errorf("'%s' is already used by the generated project for %s. Pick another name", moduleName, owner)
	}
	distribution, shadowed := shadowsInstalledModule(moduleName)
	if !shadowed {
		return nil
	}
	return fmt.Errorf(
		"'%s' becomes the Python module '%s', which is also the module installed by %s.\n"+
			"The project package would shadow it and every request would fail with "+
			"ModuleNotFoundError. Pick another name, for example '%s-app'",
		name, moduleName, distribution, name)
}
