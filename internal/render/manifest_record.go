package render

import (
	"fmt"

	"github.com/vaibdix/django-stack/internal/config"
	"github.com/vaibdix/django-stack/internal/pyenv"
)

const StackManifestFile = "django-stack.json"

type StackManifest struct {
	Schema    int              `json:"schemaVersion"`
	Tool      string           `json:"tool"`
	Version   string           `json:"version"`
	CreatedAt string           `json:"createdAt"`
	Answers   config.Answers   `json:"answers"`
	Resolved  ManifestResolved `json:"resolved"`
}

type ManifestResolved struct {
	PythonVersion  string               `json:"pythonVersion"`
	RequiresPython string               `json:"requiresPython"`
	ModuleName     string               `json:"moduleName"`
	DatabaseImage  string               `json:"databaseImage,omitzero"`
	AddedAddOns    []config.AddOn       `json:"addedAddOns,omitzero"`
	PackageManager ManifestManagers     `json:"packageManager"`
	Dependencies   ManifestDependencies `json:"dependencies"`
}

type ManifestDependencies struct {
	Python    map[string]string `json:"python"`
	PythonDev map[string]string `json:"pythonDev,omitzero"`
	Node      map[string]string `json:"node,omitzero"`
}

type ManifestManagers struct {
	Python ManifestTool `json:"python"`
	Node   ManifestTool `json:"node,omitzero"`
}

type ManifestTool struct {
	Name    string `json:"name"`
	Version string `json:"version,omitzero"`
}

func DirectVersions(specs []string, resolved map[string]string) (map[string]string, error) {
	if len(specs) == 0 {
		return nil, nil
	}

	out := make(map[string]string, len(specs))
	for _, spec := range specs {
		version, ok := installedVersion(spec, resolved)
		if !ok {
			return nil, fmt.Errorf("%s was installed but the package freeze does not list it", DistributionName(spec))
		}
		out[DistributionName(spec)] = version
	}
	return out, nil
}

func PythonSeries(selected string) string {
	major, minor, _, _, ok := pyenv.ParseDetailedVersionOutput(selected)
	if !ok {
		return ""
	}
	return fmt.Sprintf("%d.%d", major, minor)
}

func (p Project) SummaryLine() []string {
	parts := []string{"Django " + p.Cfg.Django().String(), string(p.Cfg.Stack()), string(p.Cfg.Database())}
	if addOns := p.Cfg.EnabledAddOns(); len(addOns) > 0 {
		parts = append(parts, fmt.Sprintf("%d add-ons", len(addOns)))
	}
	return parts
}
