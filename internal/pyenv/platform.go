package pyenv

import "strings"

const Windows = "windows"

func IsWindows(goos string) bool { return goos == Windows }

func VenvBin(goos string) string {
	if IsWindows(goos) {
		return "Scripts"
	}
	return "bin"
}

func ExeSuffix(goos string) string {
	if IsWindows(goos) {
		return ".exe"
	}
	return ""
}

func ScriptExt(goos string) string {
	if IsWindows(goos) {
		return ".bat"
	}
	return ".sh"
}

func InterpreterRel(goos string) string {
	separator := "/"
	if IsWindows(goos) {
		separator = `\`
	}
	return strings.Join([]string{VenvDir, VenvBin(goos), "python" + ExeSuffix(goos)}, separator)
}

func InterpreterCommand(goos string) string {
	if IsWindows(goos) {
		return InterpreterRel(goos)
	}
	return "./" + InterpreterRel(goos)
}

func pathFinder(goos string) (string, []string) {
	if IsWindows(goos) {
		return "where", nil
	}
	return "which", []string{"-a"}
}

func (h Host) knownPythonPaths() []string {
	switch h.GOOS {
	case Windows:
		return h.windowsPythonPaths()
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
