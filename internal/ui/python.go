package ui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

const minPythonVersionMajor = 3
const minPythonVersionMinor = 8

var pythonVersionPattern = regexp.MustCompile(`(?i)(\d+)\.(\d+)`)

func isPyenvActive() bool {
	cmd := exec.Command("pyenv", "version")
	output, err := cmd.Output()
	outputStr := string(output)
	if err != nil {
		return strings.Contains(outputStr, "pyenv") || strings.Contains(outputStr, "no global")
	}
	return !strings.Contains(outputStr, "no pyenv") && !strings.Contains(outputStr, "command not found")
}

func hasPyenvVersionError() bool {
	cmd := exec.Command("python", "--version")
	output, err := cmd.CombinedOutput()
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

func attemptFixPyenv() bool {
	if !isPyenvActive() {
		return false
	}

	cmd := exec.Command("pyenv", "global", "system")
	_, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}

	cmd = exec.Command("pyenv", "global")
	output, _ := cmd.Output()
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
	cmd := exec.Command(pythonPath, "--version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		cmd = exec.Command(pythonPath, "-c", "import sys; print(sys.version_info.major, sys.version_info.minor)")
		output, err = cmd.CombinedOutput()
	}

	if err != nil {
		return 0, 0, false
	}

	return parsePythonVersionOutput(string(output))
}

func parsePythonVersionOutput(output string) (int, int, bool) {
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
	if major > minPythonVersionMajor {
		return true
	}
	if major == minPythonVersionMajor && minor >= minPythonVersionMinor {
		return true
	}
	return false
}

func verifyPythonWorks(pythonPath string) bool {
	if !isPythonVersionSufficient(pythonPath) {
		return false
	}

	cmd := exec.Command(pythonPath, "-c", "print('ok')")
	output, err := cmd.CombinedOutput()
	if err != nil {
		errStr := strings.ToLower(string(output))
		if strings.Contains(errStr, "no global") ||
			strings.Contains(errStr, "pyenv") {
			return false
		}
		return false
	}

	outputStr := strings.TrimSpace(string(output))
	return outputStr == "ok"
}

func findWindowsPython(pyenvIssue bool) (string, bool) {
	if verifyPythonWorks("py") {
		return "py", true
	}

	if !pyenvIssue {
		cmd := exec.Command("where", "python")
		output, _ := cmd.CombinedOutput()
		for _, line := range strings.Split(string(output), "\n") {
			path := strings.TrimSpace(line)
			if path == "" {
				continue
			}
			pathLower := strings.ToLower(path)
			if strings.Contains(pathLower, "pyenv") || strings.Contains(pathLower, ".env") {
				continue
			}
			if verifyPythonWorks(path) {
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
				if verifyPythonWorks(pythonExe) {
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
			if verifyPythonWorks(path) {
				return path, true
			}
		}
	}

	if !pyenvIssue {
		cmd := exec.Command("which", "python3")
		output, _ := cmd.Output()
		path := strings.TrimSpace(string(output))
		if path != "" && !strings.Contains(path, ".pyenv") {
			if verifyPythonWorks(path) {
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
			if verifyPythonWorks(path) {
				return path, true
			}
		}
	}

	return "", false
}

func findLinuxPython(pyenvIssue bool) (string, bool) {
	if !pyenvIssue {
		cmd := exec.Command("which", "python3")
		output, _ := cmd.Output()
		path := strings.TrimSpace(string(output))
		if path != "" && !strings.Contains(path, ".pyenv") {
			if verifyPythonWorks(path) {
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
			if verifyPythonWorks(path) {
				return path, true
			}
		}
	}

	return "", false
}

func findWorkingPython(pyenvIssue bool) (string, bool) {
	options := discoverPythonOptions()
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
