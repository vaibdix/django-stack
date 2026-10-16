package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/vaibdix/django-cli/internal/scaffold/pyenv"
)

const (
	PythonVersionFile = ".python-version"
	RequirementsFile  = "requirements.txt"
	LockFile          = "uv.lock"
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
	return normalizeDistribution(strings.TrimSpace(name))
}

func normalizeDistribution(name string) string {
	return strings.ToLower(strings.NewReplacer("_", "-", ".", "-").Replace(name))
}

func (e *Env) freezeVersions(projectPath string) map[string]string {
	if e.frozen != nil {
		return e.frozen
	}

	ctx, cancel := e.timeoutContext()
	defer cancel()

	name, args := pyenv.FreezeCommand(projectPath)
	output, err := e.runIn(ctx, projectPath, name, args...)
	if err != nil {
		e.Step("Warning: could not read installed package versions: " + err.Error())
		return nil
	}

	versions := make(map[string]string)
	for line := range strings.SplitSeq(string(output), "\n") {
		distribution, version, ok := strings.Cut(strings.TrimSpace(line), "==")
		if !ok {
			continue
		}
		versions[normalizeDistribution(distribution)] = version
	}
	e.frozen = versions
	return versions
}

func pinSpecs(specs []string, versions map[string]string) []string {
	pinned := make([]string, 0, len(specs))
	for _, spec := range specs {
		pinned = append(pinned, pinSpec(spec, versions))
	}
	sort.Slice(pinned, func(i, j int) bool {
		return distributionName(pinned[i]) < distributionName(pinned[j])
	})
	return pinned
}

func pinSpec(spec string, versions map[string]string) string {
	if strings.ContainsAny(spec, "<>=!~") {
		return spec
	}
	version, ok := versions[distributionName(spec)]
	if !ok {
		return spec
	}
	if ceiling := nextIncompatibleVersion(version); ceiling != "" {
		return spec + ">=" + version + ",<" + ceiling
	}
	return spec + ">=" + version
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

func renderTomlArray(key string, values []string) string {
	if len(values) == 0 {
		return key + " = []\n"
	}

	var b strings.Builder
	b.WriteString(key + " = [\n")
	for _, value := range values {
		fmt.Fprintf(&b, "    %q,\n", value)
	}
	b.WriteString("]\n")
	return b.String()
}

func (e *Env) writeDependencyManifest(projectPath string) error {
	e.deps.addDev(e.declaredFormattingPackages()...)

	versions := e.freezeVersions(projectPath)

	runtime := pinSpecs(e.deps.runtime, versions)
	dev := pinSpecs(e.deps.dev, versions)

	metadata := e.pythonToolingMetadata()
	pe := NewPyprojectEditor(projectPath, e.Cfg.ProjectName, metadata)
	pe.SetProjectDependencies(runtime)
	if len(dev) > 0 {
		pe.SetDependencyGroup("dev", dev)
	}
	if err := pe.Save(); err != nil {
		return err
	}
	e.Step(fmt.Sprintf("Declared %d runtime and %d dev dependencies in %s", len(runtime), len(dev), PyprojectFile))

	if err := e.writePythonVersionFile(projectPath, metadata); err != nil {
		return err
	}

	if !pyenv.UVAvailable() {
		return e.writeRequirementsFiles(projectPath, versions)
	}
	return e.lockDependencies(projectPath)
}

func (e *Env) writePythonVersionFile(projectPath string, metadata PythonToolingMetadata) error {
	version := e.Cfg.SelectedPythonVersion
	if major, minor, patch, _, ok := pyenv.ParseDetailedVersionOutput(version); ok {
		version = fmt.Sprintf("%d.%d.%d", major, minor, patch)
	} else {
		version = strings.TrimPrefix(metadata.RequiresPython, ">=")
	}

	path := filepath.Join(projectPath, PythonVersionFile)
	if err := os.WriteFile(path, []byte(version+"\n"), 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", PythonVersionFile, err)
	}
	e.Step("Created: " + PythonVersionFile)
	return nil
}

func (e *Env) lockDependencies(projectPath string) error {
	ctx, cancel := e.timeoutContext()
	defer cancel()

	if output, err := e.runIn(ctx, projectPath, "uv", "lock"); err != nil {
		e.Step(fmt.Sprintf("Warning: uv lock failed, so %s was not written: %v\nOutput: %s", LockFile, err, string(output)))
		return nil
	}

	e.Step("Created: " + LockFile + " (run 'uv sync --frozen' to reproduce this environment)")
	return nil
}
