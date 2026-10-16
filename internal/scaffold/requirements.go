package scaffold

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/vaibdix/django-cli/internal/pypi"
	"github.com/vaibdix/django-cli/internal/scaffold/pyenv"
)

const DevRequirementsFile = "requirements-dev.txt"

const dependencyClosureScript = `
import sys
from importlib.metadata import PackageNotFoundError, distribution
from pip._vendor.packaging.requirements import Requirement
from pip._vendor.packaging.utils import canonicalize_name

seen = set()
pinned = {}
pending = [Requirement(spec) for spec in sys.argv[1:]]
while pending:
    requirement = pending.pop()
    name = canonicalize_name(requirement.name)
    key = (name, frozenset(requirement.extras))
    if key in seen:
        continue
    seen.add(key)
    try:
        dist = distribution(requirement.name)
    except PackageNotFoundError:
        continue
    pinned[name] = dist.version
    environments = [{"extra": extra} for extra in requirement.extras] + [{"extra": ""}]
    for raw in dist.requires or []:
        child = Requirement(raw)
        if child.marker is None or any(child.marker.evaluate(env) for env in environments):
            pending.append(child)

for name in sorted(pinned):
    print(name + "==" + pinned[name])
`

func (e *Env) dependencyClosure(projectPath string, specs []string) (map[string]string, error) {
	if len(specs) == 0 {
		return map[string]string{}, nil
	}

	ctx, cancel := e.timeoutContext()
	defer cancel()

	args := append([]string{"-c", dependencyClosureScript}, specs...)
	output, err := e.runIn(ctx, projectPath, pyenv.InterpreterPath(projectPath), args...)
	if err != nil {
		return nil, err
	}

	closure := map[string]string{}
	for line := range strings.SplitSeq(string(output), "\n") {
		name, version, ok := strings.Cut(strings.TrimSpace(line), "==")
		if ok {
			closure[normalizeDistribution(name)] = version
		}
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
	return pypi.FileHashes(ctx, &http.Client{Timeout: 30 * time.Second}, project, version)
}

// requirementHashes looks up every pin's published hashes. pip's hash-checking
// mode is all or nothing: once one line carries a hash, every requirement must,
// so a single failed lookup means writing none.
func (e *Env) requirementHashes(pins ...map[string]string) (map[string][]string, error) {
	ctx, cancel := e.timeoutContext()
	defer cancel()

	type result struct {
		name   string
		hashes []string
		err    error
	}
	results := make(chan result)
	count := 0
	for _, versions := range pins {
		for name, version := range versions {
			count++
			go func() {
				hashes, err := e.fileHashes(ctx, name, version)
				results <- result{name: name, hashes: hashes, err: err}
			}()
		}
	}

	hashes := make(map[string][]string, count)
	var firstErr error
	for range count {
		r := <-results
		if r.err != nil && firstErr == nil {
			firstErr = r.err
		}
		hashes[r.name] = r.hashes
	}
	if firstErr != nil {
		return nil, firstErr
	}
	return hashes, nil
}

func directPins(specs []string, versions map[string]string) map[string]string {
	pins := map[string]string{}
	for _, spec := range specs {
		name := distributionName(spec)
		if version, ok := versions[name]; ok {
			pins[name] = version
		}
	}
	return pins
}

func (e *Env) writeRequirementsFiles(projectPath string, frozen map[string]string) error {
	complete := true
	runtime, err := e.dependencyClosure(projectPath, e.deps.runtime)
	if err != nil {
		e.Step("Warning: could not resolve the runtime dependency tree, so only direct dependencies are pinned: " + err.Error())
		runtime = directPins(e.deps.runtime, frozen)
		complete = false
	}
	dev, err := e.dependencyClosure(projectPath, e.deps.dev)
	if err != nil {
		e.Step("Warning: could not resolve the dev dependency tree, so only direct dependencies are pinned: " + err.Error())
		dev = directPins(e.deps.dev, frozen)
		complete = false
	}

	var hashes map[string][]string
	if complete {
		if hashes, err = e.requirementHashes(runtime, dev); err != nil {
			e.Step("Warning: could not fetch package hashes from PyPI, so the requirements are pinned without --hash: " + err.Error())
		}
	}

	files := map[string]string{
		RequirementsFile:    pinnedLines(runtime, nil, hashes),
		DevRequirementsFile: "-r " + RequirementsFile + "\n" + pinnedLines(dev, runtime, hashes),
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(projectPath, name), []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", name, err)
		}
	}
	e.Step("Created: " + RequirementsFile + " (runtime) and " + DevRequirementsFile +
		" (tooling); uv is not installed, so no lock file was written")
	return nil
}
