package scaffold

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/vaibdix/django-stack/internal/pyenv"
	"github.com/vaibdix/django-stack/internal/pypi"
	"github.com/vaibdix/django-stack/internal/releases"
)

const devRequirementsFile = "requirements-dev.txt"

type pipReport struct {
	Version string `json:"version"`
	Install []struct {
		Metadata struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		} `json:"metadata"`
	} `json:"install"`
}

const pipReportVersion = "1"

func (e *Env) dependencyClosure(ctx context.Context, projectPath, group string, specs []string, installed map[string]string) (map[string]string, error) {
	if len(specs) == 0 {
		return map[string]string{}, nil
	}

	venv := filepath.Join(projectPath, pyenv.VenvDir)
	constraints := filepath.Join(venv, "django-stack-constraints.txt")
	report := filepath.Join(venv, "django-stack-"+group+"-report.json")
	defer func() { _ = os.Remove(constraints); _ = os.Remove(report) }()

	if err := os.WriteFile(constraints, []byte(pinnedLines(installed, nil, nil)), 0o644); err != nil {
		return nil, fmt.Errorf("failed to write the pip constraints: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, defaultCommandTimeout)
	defer cancel()

	args := append([]string{"-m", "pip", "install", "--dry-run", "--ignore-installed", "--quiet",
		"--report", report, "--constraint", constraints}, specs...)
	if _, err := e.runIn(ctx, projectPath, pyenv.InterpreterPath(projectPath), args...); err != nil {
		return nil, err
	}

	raw, err := os.ReadFile(report)
	if err != nil {
		return nil, fmt.Errorf("pip wrote no installation report: %w", err)
	}
	var parsed pipReport
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse pip's installation report: %w", err)
	}
	if parsed.Version != pipReportVersion {
		return nil, fmt.Errorf("pip's installation report is version %q; django-stack reads version %s", parsed.Version, pipReportVersion)
	}

	closure := make(map[string]string, len(parsed.Install))
	for _, item := range parsed.Install {
		closure[pypi.NormalizeName(item.Metadata.Name)] = item.Metadata.Version
	}
	return closure, nil
}

func pinnedLines(versions map[string]string, exclude map[string]string, hashes map[string][]string) string {
	names := make([]string, 0, len(versions))
	for name := range versions {
		if _, skip := exclude[name]; !skip {
			names = append(names, name)
		}
	}
	sort.Strings(names)

	var b strings.Builder
	for _, name := range names {
		b.WriteString(name + "==" + versions[name])
		for _, hash := range hashes[name] {
			b.WriteString(" \\\n    --hash=" + hash)
		}
		b.WriteString("\n")
	}
	return b.String()
}

func (e *Env) fileHashes(ctx context.Context, project, version string) ([]string, error) {
	if e.hashLookup != nil {
		return e.hashLookup(ctx, project, version)
	}
	index := releases.IndexFor(e.toolchain, os.Getenv)
	return pypi.FileHashes(ctx, &http.Client{Timeout: 30 * time.Second}, index, project, version)
}

const hashLookups = 8

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
	group.SetLimit(hashLookups)
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

func (e *Env) writeRequirementsFiles(ctx context.Context, projectPath string, installed map[string]string) error {
	runtime, err := e.dependencyClosure(ctx, projectPath, "runtime", e.deps.runtime, installed)
	if err != nil {
		return fmt.Errorf("failed to resolve the runtime dependency tree: %w", err)
	}
	dev, err := e.dependencyClosure(ctx, projectPath, "dev", e.deps.dev, installed)
	if err != nil {
		return fmt.Errorf("failed to resolve the dev dependency tree: %w", err)
	}
	hashes, err := e.requirementHashes(ctx, runtime, dev)
	if err != nil {
		return fmt.Errorf("failed to fetch the package hashes the requirements pin: %w", err)
	}

	header := e.requirementsHeader()
	files := map[string]string{
		requirementsFile:    header + pinnedLines(runtime, nil, hashes),
		devRequirementsFile: header + "-r " + requirementsFile + "\n" + pinnedLines(dev, runtime, hashes),
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(projectPath, name), []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", name, err)
		}
	}
	e.Step("Created: " + requirementsFile + " (runtime) and " + devRequirementsFile +
		" (tooling); uv is not installed, so no lock file was written")
	return nil
}

func (e *Env) requirementsHeader() string {
	return fmt.Sprintf(`# Pinned with hashes by django-stack for Python %s on %s/%s, the machine
# that generated it. pip resolves for one platform only: a dependency that
# another OS needs (Django needs tzdata on Windows) is not listed, and
# pip's hash-checking mode refuses the file there. Regenerate it on the
# deploy platform, or use uv, whose uv.lock covers every platform.
`, e.resolved.PythonVersion, runtime.GOOS, runtime.GOARCH)
}
