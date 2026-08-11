// Package pyenv discovers usable Python interpreters, creates virtual
// environments and installs packages into them.
//
// It depends only on command (and the standard library), so it is a leaf of the
// scaffolding dependency graph. Interpreter discovery in particular shells out
// a lot; routing it through a command.Runner makes it testable.
package pyenv

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

// The minimum Python version this tool supports. Django 4.2 — the oldest
// release the picker offers — requires 3.8, so anything below that could not
// produce a working project.
const (
	// MinMajor is the minimum supported Python major version.
	MinMajor = 3
	// MinMinor is the minimum supported Python minor version.
	MinMinor = 8
)

var pythonVersionPattern = regexp.MustCompile(`(?i)(\d+)\.(\d+)`)

// probeTimeout bounds each interpreter probe.
//
// Discovery execs every candidate on the machine just to read a version
// string. A single wedged interpreter — a broken pyenv shim, a stale network
// mount — would otherwise hang the picker with no way out.
const probeTimeout = 5 * time.Second

// probe runs a short-lived command with a deadline and returns its combined
// output. Every discovery subprocess goes through here.
func probe(name string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), probeTimeout)
	defer cancel()
	return exec.CommandContext(ctx, name, args...).CombinedOutput()
}

func isPyenvActive() bool {
	output, err := probe("pyenv", "version")
	outputStr := string(output)
	if err != nil {
		return strings.Contains(outputStr, "pyenv") || strings.Contains(outputStr, "no global")
	}
	return !strings.Contains(outputStr, "no pyenv") && !strings.Contains(outputStr, "command not found")
}

// HasVersionError reports whether pyenv is installed but pointing at a
// version that is not actually available — the classic broken-shim state.
func HasVersionError() bool {
	output, err := probe("python", "--version")
	outputStr := string(output)

	if strings.Contains(outputStr, "No global") ||
		strings.Contains(outputStr, "no global") ||
		strings.Contains(outputStr, "pyenv") {
		return true
	}

	if err != nil {
		errStr := err.Error()
		return strings.Contains(errStr, "No global") ||
			strings.Contains(errStr, "no global") ||
			strings.Contains(errStr, "pyenv")
	}

	return false
}

// AttemptFix tries to repair a broken pyenv shim by rehashing. It reports
// whether the interpreter works afterwards.
func AttemptFix() bool {
	if !isPyenvActive() {
		return false
	}

	if _, err := probe("pyenv", "global", "system"); err != nil {
		return false
	}

	output, _ := probe("pyenv", "global")
	return strings.TrimSpace(string(output)) == "system"
}

func warnUserAboutPyenv() {
	fmt.Println("")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("⚠️  WARNING: pyenv detected but not properly configured")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("")
	fmt.Println("To fix pyenv permanently, run one of these commands:")
	fmt.Println("")
	fmt.Println("  # Option 1: Use system Python (recommended if you have Python installed)")
	fmt.Println("  pyenv global system")
	fmt.Println("")
	fmt.Println("  # Option 2: Install and use a specific Python version")
	fmt.Println("  pyenv install 3.11.0")
	fmt.Println("  pyenv global 3.11.0")
	fmt.Println("")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("")
}

func checkPythonVersion(pythonPath string) (int, int, bool) {
	output, err := probe(pythonPath, "--version")
	if err != nil {
		// Some builds print the version to stderr or not at all; ask Python
		// itself as a fallback.
		output, err = probe(pythonPath, "-c", "import sys; print(sys.version_info.major, sys.version_info.minor)")
	}

	if err != nil {
		return 0, 0, false
	}

	return ParseVersionOutput(string(output))
}

// ParseVersionOutput extracts major and minor from `python --version` output.
// The bool reports whether parsing succeeded.
func ParseVersionOutput(output string) (int, int, bool) {
	if matches := pythonVersionPattern.FindStringSubmatch(output); len(matches) == 3 {
		major, errMajor := strconv.Atoi(matches[1])
		minor, errMinor := strconv.Atoi(matches[2])
		if errMajor == nil && errMinor == nil {
			return major, minor, true
		}
	}

	fields := strings.Fields(output)
	if len(fields) >= 2 {
		major, errMajor := strconv.Atoi(fields[0])
		minor, errMinor := strconv.Atoi(fields[1])
		if errMajor == nil && errMinor == nil {
			return major, minor, true
		}
	}

	return 0, 0, false
}

func isPythonVersionSufficient(pythonPath string) bool {
	major, minor, ok := checkPythonVersion(pythonPath)
	if !ok {
		return false
	}
	if major > MinMajor {
		return true
	}
	if major == MinMajor && minor >= MinMinor {
		return true
	}
	return false
}

// Works reports whether pythonPath is an interpreter that runs and meets the
// minimum supported version.
func Works(pythonPath string) bool {
	if !isPythonVersionSufficient(pythonPath) {
		return false
	}

	// Version alone is not enough: a pyenv shim can report a version and then
	// fail to execute anything. Run a trivial program to confirm it really works.
	output, err := probe(pythonPath, "-c", "print('ok')")
	if err != nil {
		return false
	}

	return strings.TrimSpace(string(output)) == "ok"
}

func findWindowsPython(pyenvIssue bool) (string, bool) {
	if Works("py") {
		return "py", true
	}

	if !pyenvIssue {
		output, _ := probe("where", "python")
		for line := range strings.SplitSeq(string(output), "\n") {
			path := strings.TrimSpace(line)
			if path == "" {
				continue
			}
			pathLower := strings.ToLower(path)
			if strings.Contains(pathLower, "pyenv") || strings.Contains(pathLower, ".env") {
				continue
			}
			if Works(path) {
				return path, true
			}
		}
	}

	searchPaths := []string{
		os.Getenv("LOCALAPPDATA") + "\\Programs\\Python",
		os.Getenv("USERPROFILE") + "\\AppData\\Local\\Programs\\Python",
		"C:\\Python",
		"C:\\Program Files\\Python",
		"C:\\Program Files (x86)\\Python",
	}

	for _, basePath := range searchPaths {
		if basePath == "" {
			continue
		}
		entries, err := os.ReadDir(basePath)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			name := entry.Name()
			if !strings.HasPrefix(name, "Python") {
				continue
			}
			pythonExe := filepath.Join(basePath, name, "python.exe")
			if _, err := os.Stat(pythonExe); err == nil {
				if Works(pythonExe) {
					return pythonExe, true
				}
			}
		}
	}

	return "", false
}

func findMacPython(pyenvIssue bool) (string, bool) {
	homebrewPaths := []string{
		"/opt/homebrew/bin/python3",
		"/opt/homebrew/bin/python",
	}

	for _, path := range homebrewPaths {
		if _, err := os.Stat(path); err == nil {
			if Works(path) {
				return path, true
			}
		}
	}

	if !pyenvIssue {
		output, _ := probe("which", "python3")
		path := strings.TrimSpace(string(output))
		if path != "" && !strings.Contains(path, ".pyenv") {
			if Works(path) {
				return path, true
			}
		}
	}

	systemPaths := []string{
		"/usr/local/bin/python3",
		"/usr/local/bin/python",
		"/usr/bin/python3",
		"/usr/bin/python",
		"/opt/local/bin/python3",
		"/opt/local/bin/python",
	}

	for _, path := range systemPaths {
		if _, err := os.Stat(path); err == nil {
			if Works(path) {
				return path, true
			}
		}
	}

	return "", false
}

func findLinuxPython(pyenvIssue bool) (string, bool) {
	if !pyenvIssue {
		output, _ := probe("which", "python3")
		path := strings.TrimSpace(string(output))
		if path != "" && !strings.Contains(path, ".pyenv") {
			if Works(path) {
				return path, true
			}
		}
	}

	systemPaths := []string{
		"/usr/bin/python3",
		"/usr/local/bin/python3",
		"/usr/bin/python",
		"/usr/local/bin/python",
	}

	for _, path := range systemPaths {
		if _, err := os.Stat(path); err == nil {
			if Works(path) {
				return path, true
			}
		}
	}

	return "", false
}

// FindWorking returns the first usable interpreter for the host platform.
func FindWorking(pyenvIssue bool) (string, bool) {
	options := Discover()
	if len(options) > 0 {
		return options[0].Path, true
	}

	switch runtime.GOOS {
	case "windows":
		return findWindowsPython(pyenvIssue)
	case "darwin":
		return findMacPython(pyenvIssue)
	default:
		return findLinuxPython(pyenvIssue)
	}
}

// Option is one interpreter offered to the user, already labelled for display.
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

// Load is the discovery entry point used by callers. It is a variable so tests
// can substitute a fixed interpreter list instead of probing the machine.
var Load = Discover

// Discover returns every usable interpreter found on this machine.
//
// It execs each candidate to read its version, so it is slow enough that
// callers should run it off the UI goroutine.
func Discover() []Option {
	defaultPath := detectDefaultPythonPath()
	candidates := append([]string{}, knownPythonPaths()...)
	candidates = append(candidates, listPythonPathsFromPath()...)

	return DiscoverFromCandidates(candidates, defaultPath)
}

// DiscoverFromCandidates is Discover restricted to an explicit candidate list,
// with defaultPath marked as the default. Exported for testing without
// depending on what is installed on the host.
func DiscoverFromCandidates(candidates []string, defaultPath string) []Option {
	if resolvedDefault, ok := NormalizeExecutablePath(defaultPath); ok {
		defaultPath = resolvedDefault
	}

	seen := map[string]bool{}
	options := make([]Option, 0, len(candidates))

	for _, candidate := range candidates {
		resolved, ok := NormalizeExecutablePath(candidate)
		if !ok || seen[resolved] {
			continue
		}
		seen[resolved] = true

		option, valid := buildPythonOption(resolved, sameExecutablePath(resolved, defaultPath))
		if !valid {
			continue
		}
		options = append(options, option)
	}

	sort.Slice(options, func(i, j int) bool {
		if options[i].IsDefault != options[j].IsDefault {
			return options[i].IsDefault
		}
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

func listPythonPathsFromPath() []string {
	if runtime.GOOS == "windows" {
		return runAndSplitLines("where", "python", "python3")
	}
	return runAndSplitLines("which", "-a", "python3", "python")
}

func runAndSplitLines(name string, args ...string) []string {
	output, err := probe(name, args...)
	if err != nil {
		return nil
	}
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	results := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			results = append(results, line)
		}
	}
	return results
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

func windowsPythonPaths() []string {
	paths := []string{
		filepath.Join(os.Getenv("LOCALAPPDATA"), "Programs", "Python", "Python314", "python.exe"),
		filepath.Join(os.Getenv("LOCALAPPDATA"), "Programs", "Python", "Python313", "python.exe"),
		filepath.Join(os.Getenv("LOCALAPPDATA"), "Programs", "Python", "Python312", "python.exe"),
		filepath.Join(os.Getenv("LOCALAPPDATA"), "Programs", "Python", "Python311", "python.exe"),
		filepath.Join(os.Getenv("LOCALAPPDATA"), "Programs", "Python", "Python310", "python.exe"),
		filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Local", "Programs", "Python", "Python314", "python.exe"),
		filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Local", "Programs", "Python", "Python313", "python.exe"),
		filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Local", "Programs", "Python", "Python312", "python.exe"),
		filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Local", "Programs", "Python", "Python311", "python.exe"),
		filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Local", "Programs", "Python", "Python310", "python.exe"),
	}
	results := make([]string, 0, len(paths))
	for _, path := range paths {
		if path != "" {
			results = append(results, path)
		}
	}
	return results
}

// NormalizeExecutablePath resolves path to a canonical absolute executable,
// following symlinks. The bool reports whether the path resolved at all.
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
	// filepath.Abs returns "" on error, so only adopt its result when it
	// succeeded; otherwise keep the path resolved so far.
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

func buildPythonOption(path string, isDefault bool) (Option, bool) {
	if !Works(path) {
		return Option{}, false
	}
	major, minor, patch, version, ok := detectPythonVersion(path)
	if !ok {
		return Option{}, false
	}
	label := fmt.Sprintf("Python %s (%s)", version, path)
	if isDefault {
		label += " [default]"
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

func detectPythonVersion(pythonPath string) (int, int, int, string, bool) {
	output, err := probe(pythonPath, "-c", "import sys; print(f'{sys.version_info.major}.{sys.version_info.minor}.{sys.version_info.micro}')")
	if err == nil {
		return ParseDetailedVersionOutput(string(output))
	}

	output, err = probe(pythonPath, "--version")
	if err != nil {
		return 0, 0, 0, "", false
	}
	return ParseDetailedVersionOutput(string(output))
}

// ParseDetailedVersionOutput extracts major, minor, patch and the full version
// string from `python --version` output.
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
