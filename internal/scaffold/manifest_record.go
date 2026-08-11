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
	"github.com/vaibdix/django-cli/internal/scaffold/pyenv"
)

const StackManifestFile = "django-stack.jsonc"

// Version is stamped from the binary's build info so a generated project records
// which release produced it.
var Version = "dev"

type stackManifest struct {
	Tool           string           `json:"tool"`
	Version        string           `json:"version"`
	CreatedAt      string           `json:"createdAt"`
	ProjectName    string           `json:"projectName"`
	ModuleName     string           `json:"moduleName"`
	Python         manifestPython   `json:"python"`
	Django         string           `json:"django"`
	Database       string           `json:"database"`
	Stack          string           `json:"stack"`
	AppName        string           `json:"appName,omitzero"`
	AddOns         []string         `json:"addons"`
	PackageManager manifestManagers `json:"packageManager"`
}

type manifestPython struct {
	Version        string `json:"version"`
	Interpreter    string `json:"interpreter"`
	RequiresPython string `json:"requiresPython"`
}

type manifestManagers struct {
	Python string `json:"python"`
	Node   string `json:"node,omitzero"`
}

// addOnFeatures are the choices the add-on step owns. The stack contributes its
// own features, and those are reported through the "stack" field instead.
var addOnFeatures = map[config.Feature]string{
	config.FeatureGit:           "git",
	config.FeatureRestFramework: "rest-framework",
	config.FeatureAllAuth:       "allauth",
	config.FeatureRuff:          "ruff",
	config.FeatureMypy:          "mypy",
	config.FeaturePreCommit:     "pre-commit",
	config.FeatureMultiEnv:      "multi-env",
}

func (e *Env) selectedAddOns() []string {
	enabled := map[config.Feature]bool{
		config.FeatureGit:           e.Cfg.InitializeGit,
		config.FeatureRestFramework: e.Cfg.SetupRestFramework,
		config.FeatureAllAuth:       e.Cfg.SetupAllAuth,
		config.FeatureRuff:          e.Cfg.SetupRuff,
		config.FeatureMypy:          e.Cfg.SetupMypy,
		config.FeaturePreCommit:     e.Cfg.SetupPreCommit,
		config.FeatureMultiEnv:      e.Cfg.SetupMultiEnv,
	}

	names := make([]string, 0, len(enabled))
	for feature, on := range enabled {
		if on {
			names = append(names, addOnFeatures[feature])
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

// now is a field rather than a package variable so parallel tests can pin the
// clock without racing each other.
func (e *Env) now() time.Time {
	if e.Clock == nil {
		return time.Now()
	}
	return e.Clock()
}

func (e *Env) buildStackManifest() stackManifest {
	pythonManager := "pip"
	if pyenv.UVAvailable() {
		pythonManager = "uv"
	}

	nodeManager := ""
	if e.Cfg.SetupTailwind {
		nodeManager = nodeManagerOrNpm().Name
	}

	return stackManifest{
		Tool:        "django-stack",
		Version:     Version,
		CreatedAt:   e.now().UTC().Format(time.RFC3339),
		ProjectName: e.Cfg.ProjectName,
		ModuleName:  e.Cfg.ModuleName(),
		Python: manifestPython{
			Version:        e.Cfg.SelectedPythonVersion,
			Interpreter:    e.Cfg.SelectedPythonPath,
			RequiresPython: e.pythonToolingMetadata().RequiresPython,
		},
		Django:         e.Cfg.DjangoVersion,
		Database:       string(e.Cfg.DatabaseType),
		Stack:          e.stackName(),
		AppName:        e.Cfg.AppName,
		AddOns:         e.selectedAddOns(),
		PackageManager: manifestManagers{Python: pythonManager, Node: nodeManager},
	}
}

const stackManifestPreamble = `// Written by django-stack. It records what this project was generated from,
// so the choices behind it stay legible long after the run.
//
// Nothing reads this file — edit or delete it freely. Credentials are never
// recorded here; those live in .env, which is gitignored.
`

func (e *Env) writeStackManifest(projectPath string) error {
	// Not json.MarshalIndent: it escapes >, < and & for HTML embedding, which
	// would render requires-python as "\u003e=3.13" in a file meant to be read.
	var encoded strings.Builder
	encoder := json.NewEncoder(&encoded)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(e.buildStackManifest()); err != nil {
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
