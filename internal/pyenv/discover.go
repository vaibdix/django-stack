package pyenv

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/vaibdix/django-stack/internal/command"
)

const (
	MinMajor = 3

	MinMinor = 10
)

const probeTimeout = 5 * time.Second

const maxConcurrentProbes = 8

func probe(ctx context.Context, run command.Runner, name string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, probeTimeout)
	defer cancel()
	return run.Run(ctx, "", name, args...)
}

func Works(ctx context.Context, run command.Runner, pythonPath string) bool {
	found, ok := Probe(ctx, run, pythonPath)
	return ok && supported(found.Major, found.Minor)
}

func supported(major, minor int) bool {
	return major == MinMajor && minor >= MinMinor
}

type Option struct {
	Path      string
	Version   string
	Label     string
	IsDefault bool
	Major     int
	Minor     int
	Patch     int
	Stdlib    []string
	Keywords  []string
}

var pythonSemanticVersionPattern = regexp.MustCompile(`(?i)(\d+)\.(\d+)(?:\.(\d+))?`)

type Host struct {
	GOOS     string
	LookPath func(string) (string, error)
	Stat     func(string) (fs.FileInfo, error)
	Getenv   func(string) string
}

func OS() Host {
	return Host{GOOS: runtime.GOOS, LookPath: exec.LookPath, Stat: os.Stat, Getenv: os.Getenv}
}

func Discover(ctx context.Context, run command.Runner) ([]Option, error) {
	return OS().Discover(ctx, run)
}

func DiscoverFromCandidates(ctx context.Context, run command.Runner, candidates []string, defaultPath string) []Option {
	return OS().DiscoverFromCandidates(ctx, run, candidates, defaultPath)
}

func (h Host) Discover(ctx context.Context, run command.Runner) ([]Option, error) {
	defaultPath := h.detectDefaultPythonPath()
	uvPaths, uvErr := h.uvManagedPythonPaths(ctx, run)
	pathPaths, pathErr := listPythonPathsFromPath(ctx, run, h.GOOS)
	candidates := append([]string{}, uvPaths...)
	candidates = append(candidates, h.knownPythonPaths()...)
	candidates = append(candidates, pathPaths...)

	return h.DiscoverFromCandidates(ctx, run, candidates, defaultPath), errors.Join(uvErr, pathErr)
}

func (h Host) DiscoverFromCandidates(ctx context.Context, run command.Runner, candidates []string, defaultPath string) []Option {
	if resolvedDefault, ok := h.NormalizeExecutablePath(defaultPath); ok {
		defaultPath = resolvedDefault
	}

	seen := map[string]bool{}
	var unique []string
	for _, candidate := range candidates {
		resolved, ok := h.NormalizeExecutablePath(candidate)
		if !ok || seen[resolved] {
			continue
		}
		seen[resolved] = true
		unique = append(unique, resolved)
	}

	results := make([]Option, len(unique))
	valid := make([]bool, len(unique))
	slots := make(chan struct{}, maxConcurrentProbes)
	var wg sync.WaitGroup
	for i, path := range unique {
		wg.Go(func() {
			slots <- struct{}{}
			defer func() { <-slots }()
			results[i], valid[i] = buildPythonOption(ctx, run, path, sameExecutablePath(path, defaultPath))
		})
	}
	wg.Wait()

	options := make([]Option, 0, len(unique))
	for i, option := range results {
		if valid[i] {
			options = append(options, option)
		}
	}

	sort.Slice(options, func(i, j int) bool {
		if options[i].Major != options[j].Major {
			return options[i].Major > options[j].Major
		}
		if options[i].Minor != options[j].Minor {
			return options[i].Minor > options[j].Minor
		}
		if options[i].Patch != options[j].Patch {
			return options[i].Patch > options[j].Patch
		}
		return options[i].Path < options[j].Path
	})

	return options
}

func (h Host) detectDefaultPythonPath() string {
	for _, candidate := range []string{"python3", "python"} {
		if path, err := h.LookPath(candidate); err == nil {
			if resolved, ok := h.NormalizeExecutablePath(path); ok {
				return resolved
			}
		}
	}
	return ""
}

func listPythonPathsFromPath(ctx context.Context, run command.Runner, goos string) ([]string, error) {
	finder, flags := pathFinder(goos)

	var paths []string
	for _, name := range []string{"python3", "python"} {
		output, err := probe(ctx, run, finder, append(flags, name)...)
		var exitErr *exec.ExitError
		switch {
		case errors.As(err, &exitErr) && exitErr.ExitCode() == 1:
			continue
		case err != nil:
			return paths, fmt.Errorf("%s could not search PATH for %s: %w", finder, name, err)
		}
		for line := range strings.SplitSeq(string(output), "\n") {
			if line = strings.TrimSpace(line); line != "" {
				paths = append(paths, line)
			}
		}
	}
	return paths, nil
}

const newestInstallerMinor = 15

func (h Host) windowsPythonPaths() []string {
	var paths []string
	for _, base := range []string{
		h.Getenv("LOCALAPPDATA"),
		filepath.Join(h.Getenv("USERPROFILE"), "AppData", "Local"),
	} {
		if base == "" || base == filepath.Join("AppData", "Local") {
			continue
		}
		for minor := newestInstallerMinor; minor >= MinMinor; minor-- {
			paths = append(paths, filepath.Join(base, "Programs", "Python", fmt.Sprintf("Python3%d", minor), "python.exe"))
		}
	}
	return paths
}

func (h Host) NormalizeExecutablePath(path string) (string, bool) {
	if path == "" {
		return "", false
	}
	if strings.Contains(path, string(filepath.Separator)+".venv"+string(filepath.Separator)) {
		return "", false
	}
	absPath := path
	if !filepath.IsAbs(absPath) {
		resolved, err := h.LookPath(path)
		if err != nil {
			return "", false
		}
		absPath = resolved
	}

	if resolved, err := filepath.Abs(absPath); err == nil {
		path = resolved
	} else {
		path = absPath
	}
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		path = resolved
	}
	if info, err := h.Stat(path); err != nil || info.IsDir() {
		return "", false
	}
	return path, true
}

func sameExecutablePath(left, right string) bool {
	if left == "" || right == "" {
		return false
	}
	return filepath.Clean(left) == filepath.Clean(right)
}

func buildPythonOption(ctx context.Context, run command.Runner, path string, isDefault bool) (Option, bool) {
	found, ok := Probe(ctx, run, path)
	if !ok || !supported(found.Major, found.Minor) {
		return Option{}, false
	}
	label := fmt.Sprintf("Python %s (%s)", found.Version, path)
	if isDefault {
		label += " [shell default]"
	}
	found.Path, found.Label, found.IsDefault = path, label, isDefault
	return found, true
}

const probeScript = `import json, keyword, sys; print(json.dumps({"version": "%d.%d.%d" % tuple(sys.version_info[:3]), ` +
	`"stdlib": sorted(getattr(sys, "stdlib_module_names", ())), "keywords": keyword.kwlist}))`

func Probe(ctx context.Context, run command.Runner, pythonPath string) (Option, bool) {
	output, err := probe(ctx, run, pythonPath, "-c", probeScript)
	if err != nil {
		return Option{}, false
	}
	return parseProbe(output)
}

func parseProbe(output []byte) (Option, bool) {
	var probed struct {
		Version  string   `json:"version"`
		Stdlib   []string `json:"stdlib"`
		Keywords []string `json:"keywords"`
	}
	if err := json.Unmarshal(bytes.TrimSpace(output), &probed); err != nil {
		return Option{}, false
	}
	major, minor, patch, version, ok := ParseDetailedVersionOutput(probed.Version)
	if !ok {
		return Option{}, false
	}
	return Option{Version: version, Major: major, Minor: minor, Patch: patch, Stdlib: probed.Stdlib, Keywords: probed.Keywords}, true
}

func ParseDetailedVersionOutput(output string) (int, int, int, string, bool) {
	matches := pythonSemanticVersionPattern.FindStringSubmatch(output)
	if len(matches) < 3 {
		return 0, 0, 0, "", false
	}

	major, errMajor := strconv.Atoi(matches[1])
	minor, errMinor := strconv.Atoi(matches[2])
	if errMajor != nil || errMinor != nil {
		return 0, 0, 0, "", false
	}

	patch := 0
	if len(matches) > 3 && matches[3] != "" {
		parsedPatch, err := strconv.Atoi(matches[3])
		if err != nil {
			return 0, 0, 0, "", false
		}
		patch = parsedPatch
	}

	version := fmt.Sprintf("%d.%d.%d", major, minor, patch)
	return major, minor, patch, version, true
}
