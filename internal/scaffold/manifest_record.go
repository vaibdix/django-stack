package scaffold

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/vaibdix/django-stack/internal/config"
	"github.com/vaibdix/django-stack/internal/pyenv"
)

const stackManifestFile = "django-stack.jsonc"

type stackManifest struct {
	Tool      string           `json:"tool"`
	Version   string           `json:"version"`
	CreatedAt string           `json:"createdAt"`
	Answers   config.Answers   `json:"answers"`
	Resolved  manifestResolved `json:"resolved"`
}

type manifestResolved struct {
	PythonVersion  string               `json:"pythonVersion"`
	RequiresPython string               `json:"requiresPython"`
	ModuleName     string               `json:"moduleName"`
	DatabaseImage  string               `json:"databaseImage,omitzero"`
	AddedAddOns    []config.AddOn       `json:"addedAddOns,omitzero"`
	PackageManager manifestManagers     `json:"packageManager"`
	Dependencies   manifestDependencies `json:"dependencies"`
}

type manifestDependencies struct {
	Python    map[string]string `json:"python"`
	PythonDev map[string]string `json:"pythonDev,omitzero"`
	Node      map[string]string `json:"node,omitzero"`
}

type manifestManagers struct {
	Python manifestTool `json:"python"`
	Node   manifestTool `json:"node,omitzero"`
}

type manifestTool struct {
	Name    string `json:"name"`
	Version string `json:"version,omitzero"`
}

func (e *Env) selectedAddOns() []string {
	enabled := map[config.AddOn]bool{
		config.Git:           e.cfg.InitializeGit,
		config.RestFramework: e.cfg.SetupRestFramework,
		config.Ninja:         e.cfg.SetupNinja,
		config.AllAuth:       e.cfg.SetupAllAuth,
		config.Ruff:          e.cfg.SetupRuff,
		config.Mypy:          e.cfg.SetupMypy,
		config.PreCommit:     e.cfg.SetupPreCommit,
		config.MultiEnv:      e.cfg.SetupMultiEnv,
		config.Pydantic:      e.cfg.SetupPydanticConf,
		config.Structlog:     e.cfg.SetupStructlog,
		config.Sentry:        e.cfg.SetupSentry,
	}

	names := make([]string, 0, len(enabled))
	for _, addOn := range config.AddOns() {
		if enabled[addOn] {
			names = append(names, string(addOn))
		}
	}
	sort.Strings(names)
	return names
}

func (e *Env) stackName() string {
	switch {
	case !e.cfg.CreateTemplates:
		return string(config.StackAPIOnly)
	case e.cfg.SetupTailwind && e.cfg.SetupHTMX:
		return string(config.StackTailwindHTMX)
	case e.cfg.SetupTailwind:
		return string(config.StackTailwind)
	case e.cfg.SetupHTMX:
		return string(config.StackHTMX)
	default:
		return string(config.StackStandard)
	}
}

func (e *Env) now() time.Time {
	return e.clock()
}

func directVersions(specs []string, resolved map[string]string) (map[string]string, error) {
	if len(specs) == 0 {
		return nil, nil
	}

	out := make(map[string]string, len(specs))
	for _, spec := range specs {
		version, ok := installedVersion(spec, resolved)
		if !ok {
			return nil, fmt.Errorf("%s was installed but the package freeze does not list it", distributionName(spec))
		}
		out[distributionName(spec)] = version
	}
	return out, nil
}

func (e *Env) nodeDependencies(projectPath string) (map[string]string, error) {
	if !e.cfg.SetupTailwind {
		return nil, nil
	}

	raw, err := os.ReadFile(filepath.Join(projectPath, "package.json"))
	if err != nil {
		return nil, fmt.Errorf("failed to read package.json for the stack manifest: %w", err)
	}

	var manifest struct {
		Dependencies map[string]string `json:"dependencies"`
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse package.json for the stack manifest: %w", err)
	}
	return manifest.Dependencies, nil
}

func (e *Env) pythonManager() manifestTool {
	if e.toolchain == pyenv.UV {
		return manifestTool{Name: "uv", Version: e.commandVersion("uv")}
	}
	return manifestTool{Name: "pip"}
}

func (e *Env) nodeManager() manifestTool {
	if !e.cfg.SetupTailwind {
		return manifestTool{}
	}
	return manifestTool{Name: e.node.Name, Version: e.commandVersion(e.node.Name)}
}

func (e *Env) buildStackManifest(projectPath string) (stackManifest, error) {
	resolved, err := e.freezeVersions(projectPath)
	if err != nil {
		return stackManifest{}, err
	}
	python, err := directVersions(e.deps.runtime, resolved)
	if err != nil {
		return stackManifest{}, err
	}
	pythonDev, err := directVersions(e.deps.dev, resolved)
	if err != nil {
		return stackManifest{}, err
	}
	node, err := e.nodeDependencies(projectPath)
	if err != nil {
		return stackManifest{}, err
	}

	return stackManifest{
		Tool:      "django-stack",
		Version:   e.version,
		CreatedAt: e.now().UTC().Format(time.RFC3339),
		Answers: config.Answers{
			Name:     e.cfg.ProjectName,
			Python:   pythonSeries(e.cfg.SelectedPythonVersion),
			Django:   e.cfg.DjangoVersion,
			Database: e.cfg.DatabaseType,
			Stack:    config.Stack(e.stackName()),
			App:      e.cfg.AppName,
			AddOns:   e.selectedAddOns(),
		},
		Resolved: manifestResolved{
			PythonVersion:  e.cfg.SelectedPythonVersion,
			RequiresPython: e.pythonToolingMetadata().RequiresPython,
			ModuleName:     e.cfg.ModuleName(),
			DatabaseImage:  e.cfg.DatabaseType.Spec().Image,
			AddedAddOns:    e.cfg.AddedAddOns,
			PackageManager: manifestManagers{Python: e.pythonManager(), Node: e.nodeManager()},
			Dependencies: manifestDependencies{
				Python:    python,
				PythonDev: pythonDev,
				Node:      node,
			},
		},
	}, nil
}

func pythonSeries(selected string) string {
	major, minor, _, _, ok := pyenv.ParseDetailedVersionOutput(selected)
	if !ok {
		return ""
	}
	return fmt.Sprintf("%d.%d", major, minor)
}

const stackManifestPreamble = `// Written by django-stack. "answers" holds the choices this project was
// generated from, and "resolved" what they resolved to on the day.
//
// Generate the same project again, from the folder that should hold it:
//   django-stack -config path/to/django-stack.jsonc
//
// Credentials are never recorded here; those live in .env, which is gitignored.
`

func (e *Env) writeStackManifest(projectPath string) error {
	var encoded strings.Builder
	encoder := json.NewEncoder(&encoded)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	manifest, err := e.buildStackManifest(projectPath)
	if err != nil {
		return err
	}
	if err := encoder.Encode(manifest); err != nil {
		return fmt.Errorf("failed to encode %s: %w", stackManifestFile, err)
	}

	content := stackManifestPreamble + encoded.String()
	path := filepath.Join(projectPath, stackManifestFile)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", stackManifestFile, err)
	}

	e.Step("Created: " + stackManifestFile + " (" + strings.Join(e.summaryLine(), ", ") + ")")
	return nil
}

func (e *Env) summaryLine() []string {
	parts := []string{"Django " + e.cfg.DjangoVersion, e.stackName(), string(e.cfg.DatabaseType)}
	if addOns := e.selectedAddOns(); len(addOns) > 0 {
		parts = append(parts, fmt.Sprintf("%d add-ons", len(addOns)))
	}
	return parts
}
