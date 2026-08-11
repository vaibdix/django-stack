package pyenv

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestParsePythonVersionOutput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		output string
		major  int
		minor  int
		ok     bool
	}{
		{name: "python_38", output: "Python 3.8.18", major: 3, minor: 8, ok: true},
		{name: "python_311", output: "Python 3.11.2", major: 3, minor: 11, ok: true},
		{name: "python_312", output: "Python 3.12.2", major: 3, minor: 12, ok: true},
		{name: "fallback_fields", output: "3 11\n", major: 3, minor: 11, ok: true},
		{name: "invalid", output: "unknown", major: 0, minor: 0, ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			major, minor, ok := ParseVersionOutput(tt.output)
			if major != tt.major || minor != tt.minor || ok != tt.ok {
				t.Fatalf("ParsePythonVersionOutputForTest(%q) = (%d, %d, %v), want (%d, %d, %v)", tt.output, major, minor, ok, tt.major, tt.minor, tt.ok)
			}
		})
	}
}

func TestDiscoverPythonOptionsFromCandidatesDeduplicatesAndPrefersDefault(t *testing.T) {
	t.Parallel()

	if runtime.GOOS == "windows" {
		t.Skip("test uses POSIX shell scripts")
	}

	dir := t.TempDir()
	defaultPython := writeFakePythonExecutable(t, dir, "python-default", "3.12.3")
	newerPython := writeFakePythonExecutable(t, dir, "python-newer", "3.14.3")
	aliasPath := filepath.Join(dir, "python-alias")
	if err := os.Symlink(newerPython, aliasPath); err != nil {
		t.Fatalf("failed to create symlink: %v", err)
	}

	options := DiscoverFromCandidates(
		[]string{newerPython, defaultPython, aliasPath, newerPython},
		defaultPython,
	)
	normalizedDefault, ok := NormalizeExecutablePath(defaultPython)
	if !ok {
		t.Fatalf("expected default test executable to normalize")
	}
	normalizedNewer, ok := NormalizeExecutablePath(newerPython)
	if !ok {
		t.Fatalf("expected newer test executable to normalize")
	}

	if len(options) != 2 {
		t.Fatalf("expected 2 unique Python options, got %d", len(options))
	}
	if options[0].Path != normalizedDefault {
		t.Fatalf("expected default Python to be first, got %s", options[0].Path)
	}
	if !options[0].IsDefault {
		t.Fatalf("expected first option to be marked default")
	}
	if options[1].Path != normalizedNewer {
		t.Fatalf("expected non-default newer interpreter second, got %s", options[1].Path)
	}
}

func writeFakePythonExecutable(t *testing.T, dir, name, version string) string {
	t.Helper()

	path := filepath.Join(dir, name)
	content := "#!/bin/sh\n" +
		"if [ \"$1\" = \"--version\" ]; then\n" +
		"  echo \"Python " + version + "\"\n" +
		"  exit 0\n" +
		"fi\n" +
		"if [ \"$1\" = \"-c\" ]; then\n" +
		"  case \"$2\" in\n" +
		"    *version_info.micro*) echo \"" + version + "\" ;;\n" +
		"    *print*ok*) echo \"ok\" ;;\n" +
		"    *) echo \"" + version + "\" ;;\n" +
		"  esac\n" +
		"  exit 0\n" +
		"fi\n" +
		"echo \"unsupported\" >&2\n" +
		"exit 1\n"
	if err := os.WriteFile(path, []byte(content), 0755); err != nil {
		t.Fatalf("failed to write fake python executable: %v", err)
	}
	return path
}
