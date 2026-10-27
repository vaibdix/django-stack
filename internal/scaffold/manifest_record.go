package scaffold

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/vaibdix/django-stack/internal/config"
	"github.com/vaibdix/django-stack/internal/pyenv"
	"github.com/vaibdix/django-stack/internal/render"
)

func (e *Env) now() time.Time { return e.machine().Clock() }

func (e *Env) nodeDependencies(projectPath string) (map[string]string, error) {
	if !e.Cfg.UsesTailwind() {
		return nil, nil
	}

	raw, err := e.machine().FS.ReadFile(filepath.Join(projectPath, "package.json"))
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

func (e *Env) pythonManager(ctx context.Context) render.ManifestTool {
	if e.Toolchain == pyenv.UV {
		return render.ManifestTool{Name: "uv", Version: e.commandVersion(ctx, "uv")}
	}
	return render.ManifestTool{Name: "pip"}
}

func (e *Env) nodeManager(ctx context.Context) render.ManifestTool {
	if !e.Cfg.UsesTailwind() {
		return render.ManifestTool{}
	}
	return render.ManifestTool{Name: e.Node.Name, Version: e.commandVersion(ctx, e.Node.Name)}
}

func (e *Env) buildStackManifest(ctx context.Context, projectPath string) (render.StackManifest, error) {
	resolved, err := e.freezeVersions(ctx, projectPath)
	if err != nil {
		return render.StackManifest{}, err
	}
	python, err := render.DirectVersions(e.packages.Runtime, resolved)
	if err != nil {
		return render.StackManifest{}, err
	}
	pythonDev, err := render.DirectVersions(e.packages.Dev, resolved)
	if err != nil {
		return render.StackManifest{}, err
	}
	node, err := e.nodeDependencies(projectPath)
	if err != nil {
		return render.StackManifest{}, err
	}

	answers := e.Cfg.Answers(render.PythonSeries(e.Resolved.PythonVersion))
	if !e.Resolved.IndexDate.IsZero() {
		answers.IndexDate = e.Resolved.IndexDate.UTC().Format(time.RFC3339)
	}
	return render.StackManifest{
		Schema:    config.ManifestSchema,
		Tool:      "django-stack",
		Version:   e.version,
		CreatedAt: e.now().UTC().Format(time.RFC3339),
		Answers:   answers,
		Resolved: render.ManifestResolved{
			PythonVersion:  e.Resolved.PythonVersion,
			RequiresPython: e.PythonToolingMetadata().RequiresPython,
			ModuleName:     e.Cfg.ModuleName(),
			DatabaseImage:  e.Cfg.Database().Spec().Image,
			AddedAddOns:    e.Cfg.AddedAddOns(),
			PackageManager: render.ManifestManagers{Python: e.pythonManager(ctx), Node: e.nodeManager(ctx)},
			Dependencies: render.ManifestDependencies{
				Python:    python,
				PythonDev: pythonDev,
				Node:      node,
			},
		},
	}, nil
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
		return fmt.Errorf("encode %s: %w", render.StackManifestFile, err)
	}

	if err := e.write(render.StackManifestFile, []byte(encoded.String()), render.FileMode); err != nil {
		return err
	}

	e.Step("Created: " + render.StackManifestFile + " (" + strings.Join(e.SummaryLine(), ", ") + ")")
	return nil
}
