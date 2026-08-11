package config

import "fmt"

// Modules the scaffolder installs into the project's virtualenv. A project whose
// own package has one of these names shadows the installed one: the project
// directory comes first on sys.path, so `django_htmx.middleware` resolves to the
// project package, which has no such submodule, and every request dies with
// ModuleNotFoundError.
//
// Django's own startproject refuses names that collide with an already-importable
// module, but it cannot help here — it runs before these packages are installed.
var installedModules = map[string]string{
	"django":                "Django",
	"django_htmx":           "django-htmx",
	"django_browser_reload": "django-browser-reload",
	"environ":               "django-environ",
	"rest_framework":        "djangorestframework",
	"drf_spectacular":       "drf-spectacular",
	"allauth":               "django-allauth",
	"argon2":                "argon2-cffi",
	"psycopg":               "psycopg",
	"MySQLdb":               "mysqlclient",
	"ruff":                  "ruff",
	"djlint":                "djlint",
	"mypy":                  "mypy",
}

// ShadowsInstalledModule reports the distribution a module name would shadow.
func ShadowsInstalledModule(moduleName string) (string, bool) {
	distribution, shadowed := installedModules[moduleName]
	return distribution, shadowed
}

func validateNoModuleShadowing(name, moduleName string) error {
	distribution, shadowed := ShadowsInstalledModule(moduleName)
	if !shadowed {
		return nil
	}
	return fmt.Errorf(
		"'%s' becomes the Python module '%s', which is also the module installed by %s.\n"+
			"The project package would shadow it and every request would fail with "+
			"ModuleNotFoundError. Pick another name, for example '%s-app'",
		name, moduleName, distribution, name)
}
