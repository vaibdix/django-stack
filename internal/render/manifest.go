package render

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/vaibdix/django-stack/internal/config"
	"github.com/vaibdix/django-stack/internal/pypi"
)

const (
	PythonVersionFile = ".python-version"
	RequirementsFile  = "requirements.txt"
	LockFile          = "uv.lock"
)

type DependencySet struct {
	Runtime []string
	Dev     []string
}

func (d *DependencySet) AddRuntime(specs ...string) { d.Runtime = appendUnique(d.Runtime, specs...) }

func (d *DependencySet) AddDev(specs ...string) { d.Dev = appendUnique(d.Dev, specs...) }

func appendUnique(existing []string, specs ...string) []string {
	for _, spec := range specs {
		if spec == "" {
			continue
		}
		if !containsSpec(existing, spec) {
			existing = append(existing, spec)
		}
	}
	return existing
}

func containsSpec(existing []string, spec string) bool {
	name := DistributionName(spec)
	for _, current := range existing {
		if DistributionName(current) == name {
			return true
		}
	}
	return false
}

func DistributionName(spec string) string {
	name := spec
	if idx := strings.IndexAny(name, "[<>=!~;"); idx != -1 {
		name = name[:idx]
	}
	return pypi.NormalizeName(strings.TrimSpace(name))
}

func PinSpecs(specs []string, versions map[string]string) ([]string, error) {
	pinned := make([]string, 0, len(specs))
	for _, spec := range specs {
		spec, err := pinSpec(spec, versions)
		if err != nil {
			return nil, err
		}
		pinned = append(pinned, spec)
	}
	sort.Slice(pinned, func(i, j int) bool {
		return DistributionName(pinned[i]) < DistributionName(pinned[j])
	})
	return pinned, nil
}

func pinSpec(spec string, versions map[string]string) (string, error) {
	if strings.ContainsAny(spec, "<>=!~") {
		return spec, nil
	}
	version, ok := installedVersion(spec, versions)
	if !ok {
		return "", fmt.Errorf("%s was installed but the package freeze does not list it", DistributionName(spec))
	}
	if ceiling := nextIncompatibleVersion(version); ceiling != "" {
		return spec + ">=" + version + ",<" + ceiling, nil
	}
	return spec + ">=" + version, nil
}

func installedVersion(spec string, versions map[string]string) (string, bool) {
	version, ok := versions[DistributionName(spec)]
	return version, ok
}

func nextIncompatibleVersion(version string) string {
	parts := strings.SplitN(version, ".", 3)
	if len(parts) < 2 {
		return ""
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return ""
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return ""
	}

	if major == 0 {
		return fmt.Sprintf("0.%d", minor+1)
	}
	return strconv.Itoa(major + 1)
}

func (p Project) RenderPyproject(runtime, dev []string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "[project]\nname = %s\nversion = \"0.1.0\"\nrequires-python = %s\n",
		tomlString(strings.ReplaceAll(p.Cfg.Name().String(), "-", "_")), tomlString(p.PythonToolingMetadata().RequiresPython))
	b.WriteString(renderTomlArray("dependencies", runtime))
	if p.Cfg.Has(config.Ruff) {
		b.WriteString("\n" + p.ruffSection())
	}
	if p.Cfg.Has(config.Mypy) {
		b.WriteString("\n" + p.mypySection())
	}
	if len(dev) > 0 {
		b.WriteString("\n[dependency-groups]\n" + renderTomlArray("dev", dev))
	}
	return b.String()
}

func tomlString(value string) string {
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(value) + `"`
}

func renderTomlArray(key string, values []string) string {
	if len(values) == 0 {
		return key + " = []\n"
	}

	var b strings.Builder
	b.WriteString(key + " = [\n")
	for _, value := range values {
		b.WriteString("    " + tomlString(value) + ",\n")
	}
	b.WriteString("]\n")
	return b.String()
}
