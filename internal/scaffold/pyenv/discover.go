package pyenv

import (
	"context"
	"errors"
	"fmt"
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

	"github.com/vaibdix/django-cli/internal/scaffold/command"
)

const (
	MinMajor = 3

	MinMinor = 10
)

const probeTimeout = 5 * time.Second

const maxConcurrentProbes = 8

// probe runs a short query (a version, a listing) through the runner, so it
// is cancelled with the run, killed as a process tree, and faked in tests.
func probe(ctx context.Context, run command.Runner, name string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, probeTimeout)
	defer cancel()
	return run.Run(ctx, "", name, args...)
}

// Works reports whether pythonPath runs and is a Python this generator
// supports. It is the same probe and the same rule the interpreter list uses,
// so an interpreter the picker offers is never rejected later.
func Works(ctx context.Context, run command.Runner, pythonPath string) bool {
	major, minor, _, _, ok := detectPythonVersion(ctx, run, pythonPath)
	return ok && supported(major, minor)
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
}

var pythonSemanticVersionPattern = regexp.MustCompile(`(?i)(\d+)\.(\d+)(?:\.(\d+))?`)

var Load = Discover

// Discover lists the usable interpreters on this machine. The error reports a
// source that could not be read (uv's listing, the PATH search); the interpreters found on
// PATH and in the usual install locations are still returned with it, since
// they are separate sources rather than a stand-in for uv's.
func Discover(ctx context.Context, run command.Runner) ([]Option, error) {
	defaultPath := detectDefaultPythonPath()
	uvPaths, uvErr := uvManagedPythonPaths(ctx, run)
	pathPaths, pathErr := listPythonPathsFromPath(ctx, run)
	candidates := append([]string{}, uvPaths...)
	candidates = append(candidates, knownPythonPaths()...)
	candidates = append(candidates, pathPaths...)

	return DiscoverFromCandidates(ctx, run, candidates, defaultPath), errors.Join(uvErr, pathErr)
}

func DiscoverFromCandidates(ctx context.Context, run command.Runner, candidates []string, defaultPath string) []Option {
	if resolvedDefault, ok := NormalizeExecutablePath(defaultPath); ok {
		defaultPath = resolvedDefault
	}

	seen := map[string]bool{}
	var unique []string
	for _, candidate := range candidates {
		resolved, ok := NormalizeExecutablePath(candidate)
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

func detectDefaultPythonPath() string {
	for _, candidate := range []string{"python3", "python"} {
		if path, err := exec.LookPath(candidate); err == nil {
			if resolved, ok := NormalizeExecutablePath(path); ok {
				return resolved
			}
		}
	}
	return ""
}

// listPythonPathsFromPath asks which (where on Windows) for every match of
// each name on PATH. One name at a time, because both exit 1 when any name
// has no match: asked together, a machine without a bare `python` (macOS)
// lost every python3 with it.
func listPythonPathsFromPath(ctx context.Context, run command.Runner) ([]string, error) {
	finder, flags := "which", []string{"-a"}
	if runtime.GOOS == "windows" {
		finder, flags = "where", nil
	}

	var paths []string
	for _, name := range []string{"python3", "python"} {
		output, err := probe(ctx, run, finder, append(flags, name)...)
		var exitErr *exec.ExitError
		switch {
		case errors.As(err, &exitErr) && exitErr.ExitCode() == 1:
			continue // no match for this name
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

func knownPythonPaths() []string {
	switch runtime.GOOS {
	case "windows":
		return windowsPythonPaths()
	case "darwin":
		return []string{
			"/opt/homebrew/bin/python3",
			"/opt/homebrew/bin/python",
			"/usr/local/bin/python3",
			"/usr/local/bin/python",
			"/usr/bin/python3",
			"/opt/local/bin/python3",
		}
	default:
		return []string{
			"/usr/local/bin/python3",
			"/usr/bin/python3",
			"/usr/local/bin/python",
			"/usr/bin/python",
			"/opt/bin/python3",
		}
	}
}

// newestInstallerMinor is the newest Python the Windows candidates cover. A
// newer interpreter is still found on PATH.
const newestInstallerMinor = 15

// windowsPythonPaths are the python.org installer's per-user locations. A
// base whose variable is unset is skipped: joined onto "", it would turn into
// a path relative to wherever django-stack was started.
func windowsPythonPaths() []string {
	var paths []string
	for _, base := range []string{
		os.Getenv("LOCALAPPDATA"),
		filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Local"),
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

func NormalizeExecutablePath(path string) (string, bool) {
	if path == "" {
		return "", false
	}
	if strings.Contains(path, string(filepath.Separator)+".venv"+string(filepath.Separator)) {
		return "", false
	}
	absPath := path
	if !filepath.IsAbs(absPath) {
		resolved, err := exec.LookPath(path)
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
	if info, err := os.Stat(path); err != nil || info.IsDir() {
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
	major, minor, patch, version, ok := detectPythonVersion(ctx, run, path)
	if !ok || !supported(major, minor) {
		return Option{}, false
	}
	label := fmt.Sprintf("Python %s (%s)", version, path)
	if isDefault {
		label += " [shell default]"
	}
	return Option{
		Path:      path,
		Version:   version,
		Label:     label,
		IsDefault: isDefault,
		Major:     major,
		Minor:     minor,
		Patch:     patch,
	}, true
}

const versionScript = "import sys; print(f'{sys.version_info.major}.{sys.version_info.minor}.{sys.version_info.micro}')"

func detectPythonVersion(ctx context.Context, run command.Runner, pythonPath string) (int, int, int, string, bool) {
	output, err := probe(ctx, run, pythonPath, "-c", versionScript)
	if err != nil {
		return 0, 0, 0, "", false
	}
	return ParseDetailedVersionOutput(string(output))
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
