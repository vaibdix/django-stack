package scaffold

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/vaibdix/django-cli/internal/config"
	"github.com/vaibdix/django-cli/internal/scaffold/dbdocker"
	"github.com/vaibdix/django-cli/internal/scaffold/pyenv"
)

const StackManifestFile = "django-stack.jsonc"

var Version = "dev"

type stackManifest struct {
	Tool           string               `json:"tool"`
	Version        string               `json:"version"`
	CreatedAt      string               `json:"createdAt"`
	ProjectName    string               `json:"projectName"`
	ModuleName     string               `json:"moduleName"`
	Python         manifestPython       `json:"python"`
	Django         string               `json:"django"`
	Database       manifestDatabase     `json:"database"`
	Stack          string               `json:"stack"`
	AppName        string               `json:"appName,omitzero"`
	AddOns         []string             `json:"addons"`
	PackageManager manifestManagers     `json:"packageManager"`
	Dependencies   manifestDependencies `json:"dependencies"`
}

type manifestDatabase struct {
	Engine string `json:"engine"`
	Image  string `json:"image,omitzero"`
}

type manifestDependencies struct {
	Python    map[string]string `json:"python"`
	PythonDev map[string]string `json:"pythonDev,omitzero"`
	Node      map[string]string `json:"node,omitzero"`
}

type manifestPython struct {
	Version        string `json:"version"`
	RequiresPython string `json:"requiresPython"`
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
	enabled := map[config.Feature]bool{
		config.FeatureGit:            e.Cfg.InitializeGit,
		config.FeatureRestFramework:  e.Cfg.SetupRestFramework,
		config.FeatureNinja:          e.Cfg.SetupNinja,
		config.FeatureAllAuth:        e.Cfg.SetupAllAuth,
		config.FeatureRuff:           e.Cfg.SetupRuff,
		config.FeatureMypy:           e.Cfg.SetupMypy,
		config.FeaturePreCommit:      e.Cfg.SetupPreCommit,
		config.FeatureMultiEnv:       e.Cfg.SetupMultiEnv,
		config.FeaturePydanticConfig: e.Cfg.SetupPydanticConf,
		config.FeatureStructlog:      e.Cfg.SetupStructlog,
		config.FeatureSentry:         e.Cfg.SetupSentry,
	}

	names := make([]string, 0, len(enabled))
	for feature, on := range enabled {
		if on {
			names = append(names, config.AddOnNames[feature])
		}
	}
	sort.Strings(names)
	return names
}

func (e *Env) stackName() string {
	switch {
	case !e.Cfg.CreateTemplates:
		return string(config.StackAPIOnly)
	case e.Cfg.SetupTailwind && e.Cfg.SetupHTMX:
		return string(config.StackTailwindHTMX)
	case e.Cfg.SetupTailwind:
		return string(config.StackTailwind)
	case e.Cfg.SetupHTMX:
		return string(config.StackHTMX)
	default:
		return string(config.StackStandard)
	}
}

func (e *Env) now() time.Time {
	if e.Clock == nil {
		return time.Now()
	}
	return e.Clock()
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
	if !e.Cfg.SetupTailwind {
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
		return manifestTool{Name: "uv", Version: commandVersion("uv")}
	}
	return manifestTool{Name: "pip"}
}

func (e *Env) nodeManager() manifestTool {
	if !e.Cfg.SetupTailwind {
		return manifestTool{}
	}
	return manifestTool{Name: e.node.Name, Version: commandVersion(e.node.Name)}
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
		Tool:        "django-stack",
		Version:     Version,
		CreatedAt:   e.now().UTC().Format(time.RFC3339),
		ProjectName: e.Cfg.ProjectName,
		ModuleName:  e.Cfg.ModuleName(),
		Python: manifestPython{
			Version:        e.Cfg.SelectedPythonVersion,
			RequiresPython: e.pythonToolingMetadata().RequiresPython,
		},
		Django: e.Cfg.DjangoVersion,
		Database: manifestDatabase{
			Engine: string(e.Cfg.DatabaseType),
			Image:  dbdocker.ImageFor(e.Cfg.DatabaseType),
		},
		Stack:          e.stackName(),
		AppName:        e.Cfg.AppName,
		AddOns:         e.selectedAddOns(),
		PackageManager: manifestManagers{Python: e.pythonManager(), Node: e.nodeManager()},
		Dependencies: manifestDependencies{
			Python:    python,
			PythonDev: pythonDev,
			Node:      node,
		},
	}, nil
}

const stackManifestPreamble = `// Written by django-stack. It records what this project was generated from,
// so the choices behind it stay legible long after the run.
//
// Nothing reads this file — edit or delete it freely. Credentials are never
// recorded here; those live in .env, which is gitignored.
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
		return fmt.Errorf("failed to encode %s: %w", StackManifestFile, err)
	}

	content := stackManifestPreamble + encoded.String()
	path := filepath.Join(projectPath, StackManifestFile)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", StackManifestFile, err)
	}

	e.Step("Created: " + StackManifestFile + " (" + strings.Join(e.summaryLine(), ", ") + ")")
	return nil
}

func (e *Env) summaryLine() []string {
	parts := []string{"Django " + e.Cfg.DjangoVersion, e.stackName(), string(e.Cfg.DatabaseType)}
	if addOns := e.selectedAddOns(); len(addOns) > 0 {
		parts = append(parts, fmt.Sprintf("%d add-ons", len(addOns)))
	}
	return parts
}
