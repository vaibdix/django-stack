package config

import "fmt"

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
