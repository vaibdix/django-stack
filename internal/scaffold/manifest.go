package scaffold

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/vaibdix/django-stack/internal/config"
	"github.com/vaibdix/django-stack/internal/pyenv"
	"github.com/vaibdix/django-stack/internal/pypi"
)

const (
	pythonVersionFile = ".python-version"
	requirementsFile  = "requirements.txt"
	lockFile          = "uv.lock"
)

type dependencySet struct {
	runtime []string
	dev     []string
}

func (d *dependencySet) addRuntime(specs ...string) { d.runtime = appendUnique(d.runtime, specs...) }

func (d *dependencySet) addDev(specs ...string) { d.dev = appendUnique(d.dev, specs...) }

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
	name := distributionName(spec)
	for _, current := range existing {
		if distributionName(current) == name {
			return true
		}
	}
	return false
}

func distributionName(spec string) string {
	name := spec
	if idx := strings.IndexAny(name, "[<>=!~;"); idx != -1 {
		name = name[:idx]
	}
	return pypi.NormalizeName(strings.TrimSpace(name))
}

func (e *Env) freezeVersions(ctx context.Context, projectPath string) (map[string]string, error) {
	if e.frozen != nil {
		return e.frozen, nil
	}

	ctx, cancel := context.WithTimeout(ctx, defaultCommandTimeout)
	defer cancel()

	name, args := pyenv.FreezeCommand(e.toolchain, projectPath)
	output, err := e.runIn(ctx, projectPath, name, args...)
	if err != nil {
		return nil, fmt.Errorf("read the installed package versions: %w", err)
	}

	versions := make(map[string]string)
	for line := range strings.SplitSeq(string(output), "\n") {
		distribution, version, ok := strings.Cut(strings.TrimSpace(line), "==")
		if !ok {
			continue
		}
		versions[pypi.NormalizeName(distribution)] = version
	}
	e.frozen = versions
	return versions, nil
}

func pinSpecs(specs []string, versions map[string]string) ([]string, error) {
	pinned := make([]string, 0, len(specs))
	for _, spec := range specs {
		spec, err := pinSpec(spec, versions)
		if err != nil {
			return nil, err
		}
		pinned = append(pinned, spec)
	}
	sort.Slice(pinned, func(i, j int) bool {
		return distributionName(pinned[i]) < distributionName(pinned[j])
	})
	return pinned, nil
}

func pinSpec(spec string, versions map[string]string) (string, error) {
	if strings.ContainsAny(spec, "<>=!~") {
		return spec, nil
	}
	version, ok := installedVersion(spec, versions)
	if !ok {
		return "", fmt.Errorf("%s was installed but the package freeze does not list it", distributionName(spec))
	}
	if ceiling := nextIncompatibleVersion(version); ceiling != "" {
		return spec + ">=" + version + ",<" + ceiling, nil
	}
	return spec + ">=" + version, nil
}

func installedVersion(spec string, versions map[string]string) (string, bool) {
	version, ok := versions[distributionName(spec)]
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

func (p project) renderPyproject(runtime, dev []string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "[project]\nname = %s\nversion = \"0.1.0\"\nrequires-python = %s\n",
		tomlString(strings.ReplaceAll(p.cfg.Name, "-", "_")), tomlString(p.pythonToolingMetadata().RequiresPython))
	b.WriteString(renderTomlArray("dependencies", runtime))
	if p.cfg.Has(config.Ruff) {
		b.WriteString("\n" + p.ruffSection())
	}
	if p.cfg.Has(config.Mypy) {
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

func (e *Env) writeDependencyManifest(ctx context.Context) error {
	e.packages.addDev(e.declaredFormattingPackages()...)

	versions, err := e.freezeVersions(ctx, e.dir)
	if err != nil {
		return err
	}
	runtime, err := pinSpecs(e.packages.runtime, versions)
	if err != nil {
		return err
	}
	dev, err := pinSpecs(e.packages.dev, versions)
	if err != nil {
		return err
	}

	if err := e.write(pyprojectFile, []byte(e.renderPyproject(runtime, dev)), fileMode); err != nil {
		return err
	}
	e.Step(fmt.Sprintf("Declared %d runtime and %d dev dependencies in %s", len(runtime), len(dev), pyprojectFile))

	if err := e.writePythonVersionFile(); err != nil {
		return err
	}

	if e.toolchain == pyenv.Pip {
		return e.writeRequirementsFiles(ctx, versions)
	}
	return e.lockDependencies(ctx, e.dir)
}

func (e *Env) writePythonVersionFile() error {
	_, _, _, version, ok := pyenv.ParseDetailedVersionOutput(e.resolved.PythonVersion)
	if !ok {
		return fmt.Errorf("cannot write %s: the selected interpreter's version %q is not major.minor.micro",
			pythonVersionFile, e.resolved.PythonVersion)
	}

	if err := e.write(pythonVersionFile, []byte(version+"\n"), fileMode); err != nil {
		return err
	}
	e.Step("Created: " + pythonVersionFile)
	return nil
}

func (e *Env) lockDependencies(ctx context.Context, projectPath string) error {
	ctx, cancel := context.WithTimeout(ctx, defaultCommandTimeout)
	defer cancel()

	if output, err := e.runIn(ctx, projectPath, "uv", "lock"); err != nil {
		return fmt.Errorf("uv lock failed, so %s could not be written: %w\nOutput: %s", lockFile, err, string(output))
	}

	e.Step("Created: " + lockFile + " (run 'uv sync --frozen' to reproduce this environment)")
	return nil
}
