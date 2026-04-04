package ui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"

	"github.com/charmbracelet/huh"
)

type PythonOption struct {
	Path      string
	Version   string
	Label     string
	IsDefault bool
	Major     int
	Minor     int
	Patch     int
}

var pythonSemanticVersionPattern = regexp.MustCompile(`(?i)(\d+)\.(\d+)(?:\.(\d+))?`)
var loadPythonOptions = discoverPythonOptions

func (m *Model) RefreshPythonOptions() {
	m.AvailablePythonOptions = loadPythonOptions()
	if len(m.AvailablePythonOptions) == 0 {
		m.SelectedPythonPath = ""
		m.SelectedPythonVersion = ""
		m.PythonForm = buildUnavailablePythonForm(&m.SelectedPythonPath)
		return
	}

	if m.SelectedPythonPath == "" || !m.hasPythonOption(m.SelectedPythonPath) {
		m.SelectedPythonPath = m.AvailablePythonOptions[0].Path
	}
	m.syncSelectedPythonOption()
	m.PythonForm = m.buildPythonForm()
}

func (m *Model) buildPythonForm() *huh.Form {
	options := make([]huh.Option[string], 0, len(m.AvailablePythonOptions))
	for _, option := range m.AvailablePythonOptions {
		options = append(options, huh.NewOption(option.Label, option.Path))
	}

	return huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Python Interpreter").
				Description("Choose the Python runtime used for the virtualenv and Django project setup.").
				Options(options...).
				Value(&m.SelectedPythonPath),
		),
	)
}

func buildUnavailablePythonForm(target *string) *huh.Form {
	if target != nil {
		*target = ""
	}
	return huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Python Interpreter").
				Description("No supported Python 3.8+ interpreter was detected on this machine.").
				Options(huh.NewOption("No Python 3.8+ interpreter found", "")).
				Value(target),
		),
	)
}

func (m *Model) hasPythonOption(path string) bool {
	for _, option := range m.AvailablePythonOptions {
		if option.Path == path {
			return true
		}
	}
	return false
}

func (m *Model) syncSelectedPythonOption() {
	for _, option := range m.AvailablePythonOptions {
		if option.Path == m.SelectedPythonPath {
			m.SelectedPythonVersion = option.Version
			return
		}
	}
	if len(m.AvailablePythonOptions) > 0 {
		m.SelectedPythonPath = m.AvailablePythonOptions[0].Path
		m.SelectedPythonVersion = m.AvailablePythonOptions[0].Version
		return
	}
	m.SelectedPythonVersion = ""
}

func (m *Model) selectedPythonOption() (PythonOption, bool) {
	for _, option := range m.AvailablePythonOptions {
		if option.Path == m.SelectedPythonPath {
			return option, true
		}
	}
	return PythonOption{}, false
}

func discoverPythonOptions() []PythonOption {
	defaultPath := detectDefaultPythonPath()
	candidates := append([]string{}, knownPythonPaths()...)
	candidates = append(candidates, listPythonPathsFromPath()...)

	return discoverPythonOptionsFromCandidates(candidates, defaultPath)
}

func discoverPythonOptionsFromCandidates(candidates []string, defaultPath string) []PythonOption {
	if resolvedDefault, ok := normalizeExecutablePath(defaultPath); ok {
		defaultPath = resolvedDefault
	}

	seen := map[string]bool{}
	options := make([]PythonOption, 0, len(candidates))

	for _, candidate := range candidates {
		resolved, ok := normalizeExecutablePath(candidate)
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
			if resolved, ok := normalizeExecutablePath(path); ok {
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
	cmd := exec.Command(name, args...)
	output, err := cmd.Output()
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

func normalizeExecutablePath(path string) (string, bool) {
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
	absPath, err := filepath.Abs(absPath)
	if err == nil {
		path = absPath
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

func buildPythonOption(path string, isDefault bool) (PythonOption, bool) {
	if !verifyPythonWorks(path) {
		return PythonOption{}, false
	}
	major, minor, patch, version, ok := detectPythonVersion(path)
	if !ok {
		return PythonOption{}, false
	}
	label := fmt.Sprintf("Python %s (%s)", version, path)
	if isDefault {
		label += " [default]"
	}
	return PythonOption{
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
	cmd := exec.Command(pythonPath, "-c", "import sys; print(f'{sys.version_info.major}.{sys.version_info.minor}.{sys.version_info.micro}')")
	output, err := cmd.CombinedOutput()
	if err == nil {
		return parseDetailedPythonVersionOutput(string(output))
	}

	cmd = exec.Command(pythonPath, "--version")
	output, err = cmd.CombinedOutput()
	if err != nil {
		return 0, 0, 0, "", false
	}
	return parseDetailedPythonVersionOutput(string(output))
}

func parseDetailedPythonVersionOutput(output string) (int, int, int, string, bool) {
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
