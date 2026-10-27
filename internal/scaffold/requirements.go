package scaffold

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"path/filepath"
	"sync"

	"github.com/vaibdix/django-stack/internal/pyenv"
	"github.com/vaibdix/django-stack/internal/pypi"
	"github.com/vaibdix/django-stack/internal/render"
	"golang.org/x/sync/errgroup"
)

func (e *Env) dependencyClosure(ctx context.Context, group string, specs []string, installed map[string]string) (map[string]string, error) {
	if len(specs) == 0 {
		return map[string]string{}, nil
	}

	projectPath := e.dir
	venv := filepath.Join(projectPath, pyenv.VenvDir)
	constraints := filepath.Join(venv, "django-stack-constraints.txt")
	report := filepath.Join(venv, "django-stack-"+group+"-report.json")
	defer func() { _ = e.machine().FS.RemoveAll(constraints); _ = e.machine().FS.RemoveAll(report) }()

	if err := e.write(filepath.Join(pyenv.VenvDir, "django-stack-constraints.txt"), []byte(render.PinnedLines(installed, nil, nil)), render.FileMode); err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, defaultCommandTimeout)
	defer cancel()

	args := append([]string{"-m", "pip", "install", "--dry-run", "--ignore-installed", "--quiet",
		"--report", report, "--constraint", constraints}, specs...)
	if _, err := e.runIn(ctx, projectPath, pyenv.InterpreterPath(projectPath), args...); err != nil {
		return nil, err
	}

	raw, err := e.machine().FS.ReadFile(report)
	if err != nil {
		return nil, fmt.Errorf("pip wrote no installation report: %w", err)
	}
	var parsed render.PipReport
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("parse pip's installation report: %w", err)
	}
	if parsed.Version != render.PipReportVersion {
		return nil, fmt.Errorf("pip's installation report is version %q; django-stack reads version %s", parsed.Version, render.PipReportVersion)
	}

	closure := make(map[string]string, len(parsed.Install))
	for _, item := range parsed.Install {
		closure[pypi.NormalizeName(item.Metadata.Name)] = item.Metadata.Version
	}
	return closure, nil
}

func (e *Env) fileHashes(ctx context.Context, project, version string) ([]string, error) {
	return e.machine().Index.FileHashes(ctx, e.indexFor(), project, version)
}

func (e *Env) requirementHashes(ctx context.Context, pins ...map[string]string) (map[string][]string, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultCommandTimeout)
	defer cancel()

	versions := map[string]string{}
	for _, set := range pins {
		maps.Copy(versions, set)
	}

	var mu sync.Mutex
	hashes := make(map[string][]string, len(versions))
	group, ctx := errgroup.WithContext(ctx)
	group.SetLimit(render.HashLookups)
	for name, version := range versions {
		group.Go(func() error {
			found, err := e.fileHashes(ctx, name, version)
			if err != nil {
				return fmt.Errorf("%s==%s: %w", name, version, err)
			}
			mu.Lock()
			hashes[name] = found
			mu.Unlock()
			return nil
		})
	}
	if err := group.Wait(); err != nil {
		return nil, err
	}
	return hashes, nil
}

func (e *Env) writeRequirementsFiles(ctx context.Context, installed map[string]string) error {
	runtime, err := e.dependencyClosure(ctx, "runtime", e.packages.Runtime, installed)
	if err != nil {
		return fmt.Errorf("resolve the runtime dependency tree: %w", err)
	}
	dev, err := e.dependencyClosure(ctx, "dev", e.packages.Dev, installed)
	if err != nil {
		return fmt.Errorf("resolve the dev dependency tree: %w", err)
	}
	hashes, err := e.requirementHashes(ctx, runtime, dev)
	if err != nil {
		return fmt.Errorf("fetch the package hashes the requirements pin: %w", err)
	}

	header := e.RequirementsHeader()
	files := map[string]string{
		render.RequirementsFile:    header + render.PinnedLines(runtime, nil, hashes),
		render.DevRequirementsFile: header + "-r " + render.RequirementsFile + "\n" + render.PinnedLines(dev, runtime, hashes),
	}
	if err := e.writeFiles(".", files); err != nil {
		return err
	}
	e.Step("Created: " + render.RequirementsFile + " (runtime) and " + render.DevRequirementsFile +
		" (tooling); uv is not installed, so no lock file was written")
	return nil
}
