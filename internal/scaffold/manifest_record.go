package scaffold

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vaibdix/django-stack/internal/config"
	"github.com/vaibdix/django-stack/internal/pyenv"
)

const stackManifestFile = "django-stack.json"

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

func (e *Env) now() time.Time {
	if e.clock == nil {
		return time.Now()
	}
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

func (p project) nodeDependencies(projectPath string) (map[string]string, error) {
	if !p.cfg.UsesTailwind() {
		return nil, nil
	}

	raw, err := os.ReadFile(filepath.Join(projectPath, "package.json"))
	if err != nil {
		return nil, fmt.Errorf("read package.json for the stack manifest: %w", err)
	}

	var manifest struct {
		Dependencies map[string]string `json:"dependencies"`
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return nil, fmt.Errorf("parse package.json for the stack manifest: %w", err)
	}
	return manifest.Dependencies, nil
}

func (e *Env) pythonManager(ctx context.Context) manifestTool {
	if e.toolchain == pyenv.UV {
		return manifestTool{Name: "uv", Version: e.commandVersion(ctx, "uv")}
	}
	return manifestTool{Name: "pip"}
}

func (e *Env) nodeManager(ctx context.Context) manifestTool {
	if !e.cfg.UsesTailwind() {
		return manifestTool{}
	}
	return manifestTool{Name: e.node.Name, Version: e.commandVersion(ctx, e.node.Name)}
}

func (e *Env) buildStackManifest(ctx context.Context, projectPath string) (stackManifest, error) {
	resolved, err := e.freezeVersions(ctx, projectPath)
	if err != nil {
		return stackManifest{}, err
	}
	python, err := directVersions(e.packages.runtime, resolved)
	if err != nil {
		return stackManifest{}, err
	}
	pythonDev, err := directVersions(e.packages.dev, resolved)
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
		Answers:   e.cfg.Answers(pythonSeries(e.resolved.PythonVersion)),
		Resolved: manifestResolved{
			PythonVersion:  e.resolved.PythonVersion,
			RequiresPython: e.pythonToolingMetadata().RequiresPython,
			ModuleName:     e.cfg.ModuleName(),
			DatabaseImage:  e.cfg.Database.Spec().Image,
			AddedAddOns:    e.cfg.AddedAddOns(),
			PackageManager: manifestManagers{Python: e.pythonManager(ctx), Node: e.nodeManager(ctx)},
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

func (e *Env) writeStackManifest(ctx context.Context) error {
	var encoded strings.Builder
	encoder := json.NewEncoder(&encoded)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	manifest, err := e.buildStackManifest(ctx, e.dir)
	if err != nil {
		return err
	}
	if err := encoder.Encode(manifest); err != nil {
		return fmt.Errorf("encode %s: %w", stackManifestFile, err)
	}

	if err := e.write(stackManifestFile, []byte(encoded.String()), fileMode); err != nil {
		return err
	}

	e.Step("Created: " + stackManifestFile + " (" + strings.Join(e.summaryLine(), ", ") + ")")
	return nil
}

func (p project) summaryLine() []string {
	parts := []string{"Django " + p.cfg.Django, string(p.cfg.Stack), string(p.cfg.Database)}
	if addOns := p.cfg.EnabledAddOns(); len(addOns) > 0 {
		parts = append(parts, fmt.Sprintf("%d add-ons", len(addOns)))
	}
	return parts
}
