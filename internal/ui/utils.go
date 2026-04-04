package ui

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"django-stack/internal/template"
	"django-stack/internal/templates"
)

func getPythonPath(projectPath string) string {
	if runtime.GOOS == "windows" {
		return filepath.Join(projectPath, VenvDir, "Scripts", "python.exe")
	}
	return filepath.Join(projectPath, VenvDir, "bin", "python")
}

func isCommandAvailable(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func isUvAvailable() bool {
	if !isCommandAvailable("uv") {
		return false
	}
	cmd := exec.Command("uv", "--version")
	if output, err := cmd.CombinedOutput(); err != nil {
		return false
	} else {
		return len(output) > 0
	}
}

func packageInstallCommand(projectPath string) (string, []string) {
	pythonPath := getPythonPath(projectPath)
	if isUvAvailable() {
		return "uv", []string{"pip", "install", "--python", pythonPath}
	}
	return pythonPath, []string{"-m", "pip", "install"}
}

func installPackages(projectPath string, packages ...string) *exec.Cmd {
	command, baseArgs := packageInstallCommand(projectPath)
	args := append([]string{}, baseArgs...)
	args = append(args, packages...)
	if command != "uv" && runtime.GOOS == "windows" {
		args = append(args, "--progress-bar", "pretty")
	}
	cmd := exec.Command(command, args...)
	cmd.Dir = projectPath
	return cmd
}

func writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0644)
}

func GetBaseCSS() string {
	return template.GetBaseCSS()
}

func GetHomeCSS() string {
	return template.GetHomeCSS()
}

func GetTemplateContent(templateType string, config map[string]string) string {
	return templates.TemplateContent(templateType, config)
}

func boolToString(b bool) string {
	if b {
		return "true"
	}
	return ""
}
