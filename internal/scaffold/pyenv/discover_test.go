package pyenv

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
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

func TestDiscoverPythonOptionsFromCandidatesDeduplicatesAndPrefersTheNewest(t *testing.T) {
	t.Parallel()

	if runtime.GOOS == "windows" {
		t.Skip("test uses POSIX shell scripts")
	}

	dir := t.TempDir()
	defaultPython := writeFakePythonExecutable(t, dir, "python-default", "3.9.6")
	newerPython := writeFakePythonExecutable(t, dir, "python-newer", "3.14.3")
	aliasPath := filepath.Join(dir, "python-alias")
	if err := os.Symlink(newerPython, aliasPath); err != nil {
		t.Fatalf("failed to create symlink: %v", err)
	}

	options := DiscoverFromCandidates(
		[]string{defaultPython, newerPython, aliasPath, newerPython},
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
	if options[0].Path != normalizedNewer {
		t.Fatalf("the newest interpreter must be preselected, got %s; "+
			"preferring the shell default caps a new project at whatever Django that old Python supports",
			options[0].Path)
	}
	if options[1].Path != normalizedDefault {
		t.Fatalf("expected the older shell default second, got %s", options[1].Path)
	}
	if !options[1].IsDefault {
		t.Fatalf("the shell default must still be marked so the user can recognise it")
	}
	if !strings.Contains(options[1].Label, "shell default") {
		t.Fatalf("expected the shell default to be labelled, got %q", options[1].Label)
	}
}

func TestUVManagedPythonsAreParsedFromJSON(t *testing.T) {
	t.Parallel()

	const listing = `[
	  {"key":"cpython-3.13.13-macos-aarch64-none","version":"3.13.13",
	   "path":"/home/u/.local/bin/python3.13","implementation":"cpython","variant":"default"},
	  {"key":"cpython-3.14.6+freethreaded-macos-aarch64-none","version":"3.14.6",
	   "path":"/home/u/.local/share/uv/python/ft/bin/python3.14","implementation":"cpython","variant":"freethreaded"},
	  {"key":"pypy-3.11.15-macos-aarch64-none","version":"3.11.15",
	   "path":"/home/u/.local/share/uv/python/pypy/bin/pypy","implementation":"pypy","variant":"default"},
	  {"key":"cpython-3.15.0b3-macos-aarch64-none","version":"3.15.0b3",
	   "path":"","implementation":"cpython","variant":"default"}
	]`

	var entries []uvPythonEntry
	if err := json.Unmarshal([]byte(listing), &entries); err != nil {
		t.Fatalf("parsing the uv listing: %v", err)
	}

	var usable []string
	for _, entry := range entries {
		if entry.usable() {
			usable = append(usable, entry.Path)
		}
	}

	if len(usable) != 1 || usable[0] != "/home/u/.local/bin/python3.13" {
		t.Fatalf("expected only the default-variant CPython with a path, got %v", usable)
	}
}

func TestUVTextListingIsParsedWhenJSONIsUnavailable(t *testing.T) {
	t.Parallel()

	const listing = `cpython-3.13.13-macos-aarch64-none    /home/u/.local/bin/python3.13 -> /home/u/.local/share/uv/python/cpython-3.13/bin/python3.13
cpython-3.9.6-macos-aarch64-none      /usr/bin/python3
cpython-3.15.0b3-macos-aarch64-none   <download available>
cpython-3.14.6+freethreaded-macos-aarch64-none  /home/u/ft/bin/python3.14
pypy-3.11.15-macos-aarch64-none       /home/u/pypy/bin/pypy`

	got := parseUVTextListing(listing)
	want := []string{"/home/u/.local/bin/python3.13", "/usr/bin/python3"}

	if len(got) != len(want) {
		t.Fatalf("parsed %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("path %d = %q, want %q", i, got[i], want[i])
		}
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
