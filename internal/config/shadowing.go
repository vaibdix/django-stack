package config

import (
	"fmt"

	"github.com/vaibdix/django-stack/internal/catalog"
)

var (
	installedModules   = catalog.Load().InstalledModules
	scaffoldOwnedNames = catalog.Load().ScaffoldNames
)

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
