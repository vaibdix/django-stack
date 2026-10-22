package pyenv

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/vaibdix/django-stack/internal/command"
	"github.com/vaibdix/django-stack/internal/command/commandtest"
	"github.com/vaibdix/django-stack/internal/report"
)

func TestWorksAppliesTheSameRuleAsThePicker(t *testing.T) {
	t.Parallel()

	if IsWindows(runtime.GOOS) {
		t.Skip("test uses POSIX shell scripts")
	}

	dir := t.TempDir()
	for version, want := range map[string]bool{"3.9.6": false, "3.10.0": true, "3.14.3": true, "4.0.0": false} {
		python := writeFakePythonExecutable(t, dir, "python-"+version, version)
		if got := Works(context.Background(), command.NewOS(), python); got != want {
			t.Errorf("Works(Python %s) = %v, want %v", version, got, want)
		}
		_, offered := buildPythonOption(context.Background(), command.NewOS(), python, false)
		if offered != want {
			t.Errorf("the picker offers Python %s: %v, but Works says %v", version, offered, want)
		}
	}
}

func TestDiscoverPythonOptionsFromCandidatesDeduplicatesAndPrefersTheNewest(t *testing.T) {
	t.Parallel()

	if IsWindows(runtime.GOOS) {
		t.Skip("test uses POSIX shell scripts")
	}

	dir := t.TempDir()
	defaultPython := writeFakePythonExecutable(t, dir, "python-default", "3.12.6")
	newerPython := writeFakePythonExecutable(t, dir, "python-newer", "3.14.3")
	aliasPath := filepath.Join(dir, "python-alias")
	if err := os.Symlink(newerPython, aliasPath); err != nil {
		t.Fatalf("failed to create symlink: %v", err)
	}

	options := DiscoverFromCandidates(
		context.Background(), command.NewOS(),
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
		"    *stdlib_module_names*) echo '{\"version\": \"" + version + "\", \"stdlib\": [\"os\"], \"keywords\": [\"class\"]}' ;;\n" +
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

type stepRecorder struct{ steps []string }

func (r *stepRecorder) Step(message string) { r.steps = append(r.steps, message) }
func (r *stepRecorder) Progress(string)     {}

func fakePython(fake *commandtest.Fake, version string) string {
	python := "/opt/python/bin/python3"
	fake.Responses[python+" -c "+probeScript] = []byte(`{"version": "` + version + `", "stdlib": ["os", "sys"], "keywords": ["class"]}` + "\n")
	return python
}

func TestAFailedUVVenvFallsBackToPipForTheWholeRun(t *testing.T) {
	t.Parallel()

	projectPath := t.TempDir()
	fake := commandtest.NewFake()
	python := fakePython(fake, "3.13.13")
	fake.FailWithPrefix("uv venv", "uv: network unreachable")
	rep := &stepRecorder{}

	got, err := CreateVenv(context.Background(), fake, rep, projectPath, python, UV)
	if err != nil {
		t.Fatalf("CreateVenv: %v", err)
	}
	if got != Pip {
		t.Errorf("after uv failed the run continues on %v, want pip", got)
	}
	want := python + " -m venv " + VenvDir
	if !fake.Ran(want) {
		t.Errorf("the fallback never ran %q; calls: %v", want, fake.CommandLines())
	}
	if !strings.Contains(strings.Join(rep.steps, "\n"), "using pip for this project") {
		t.Errorf("the switch to pip was not reported: %q", rep.steps)
	}
}

func TestAFailedPipVenvIsFatal(t *testing.T) {
	t.Parallel()

	fake := commandtest.NewFake()
	python := fakePython(fake, "3.13.13")
	fake.FailWithPrefix(python+" -m venv", "ensurepip is not available")

	if _, err := CreateVenv(context.Background(), fake, report.Nop{}, t.TempDir(), python, Pip); err == nil {
		t.Fatal("a failed python -m venv was not reported as an error")
	}
	if fake.Ran("uv venv --python " + python + " " + VenvDir) {
		t.Error("a pip run reached for uv")
	}
}

func TestAPathWithoutABarePythonStillListsItsPython3(t *testing.T) {
	t.Parallel()

	fake := commandtest.NewFake()
	finder := "which -a"
	if IsWindows(runtime.GOOS) {
		finder = "where"
	}
	fake.Responses[finder+" python3"] = []byte("/usr/bin/python3\n")
	fake.Errors[finder+" python"] = &exec.ExitError{ProcessState: exitedWith(t, 1)}

	paths, err := listPythonPathsFromPath(context.Background(), fake)
	if err != nil {
		t.Fatalf("listPythonPathsFromPath: %v", err)
	}
	if len(paths) != 1 || paths[0] != "/usr/bin/python3" {
		t.Errorf("paths = %v, want python3 kept although no bare python exists", paths)
	}
}

func TestAPathSearchThatCannotRunIsReported(t *testing.T) {
	t.Parallel()

	fake := commandtest.NewFake()
	fake.FailWithPrefix("which", "executable file not found in $PATH")
	fake.FailWithPrefix("where", "executable file not found in %PATH%")
	if _, err := listPythonPathsFromPath(context.Background(), fake); err == nil {
		t.Error("a PATH search that could not run was taken as no interpreters")
	}
}

func exitedWith(t *testing.T, code int) *os.ProcessState {
	t.Helper()
	shell, arg := "sh", "-c"
	if IsWindows(runtime.GOOS) {
		shell, arg = "cmd", "/c"
	}
	cmd := exec.CommandContext(t.Context(), shell, arg, "exit "+strconv.Itoa(code))
	_ = cmd.Run()
	return cmd.ProcessState
}

func TestWindowsCandidatesNeverBecomeRelativePaths(t *testing.T) {
	t.Setenv("LOCALAPPDATA", "")
	t.Setenv("USERPROFILE", "")
	if paths := windowsPythonPaths(); len(paths) != 0 {
		t.Errorf("with no profile variables the candidates were %v, which resolve against the working directory", paths)
	}

	t.Setenv("LOCALAPPDATA", filepath.Join(string(filepath.Separator), "Users", "u", "AppData", "Local"))
	for _, path := range windowsPythonPaths() {
		if !filepath.IsAbs(path) {
			t.Errorf("candidate %q is not absolute", path)
		}
	}
}

func TestTheProbeReadsTheInterpretersOwnModuleAndKeywordLists(t *testing.T) {
	t.Parallel()

	output := []byte(`{"version": "3.10.18", "stdlib": ["asyncio", "os", "telnetlib"], "keywords": ["False", "class", "yield"]}` + "\n")
	probed, ok := parseProbe(output)
	if !ok {
		t.Fatal("a valid probe line was rejected")
	}
	if probed.Major != 3 || probed.Minor != 10 || probed.Patch != 18 || probed.Version != "3.10.18" {
		t.Errorf("version = %d.%d.%d (%s), want 3.10.18", probed.Major, probed.Minor, probed.Patch, probed.Version)
	}
	if !slices.Contains(probed.Stdlib, "telnetlib") || !slices.Contains(probed.Keywords, "class") {
		t.Errorf("stdlib %v, keywords %v", probed.Stdlib, probed.Keywords)
	}

	for _, bad := range []string{"", "3.10.18", `{"version": "x"}`} {
		if _, ok := parseProbe([]byte(bad)); ok {
			t.Errorf("parseProbe(%q) accepted output that is not the probe's", bad)
		}
	}
}

func TestTheProbeRunsOnTheOldestAndNewestSupportedPython(t *testing.T) {
	if os.Getenv("DJANGO_STACK_E2E") == "" {
		t.Skip("set DJANGO_STACK_E2E=1 to run the probe against real interpreters")
	}
	for _, version := range []string{"3.10", "3.14"} {
		path, err := exec.CommandContext(t.Context(), "uv", "python", "find", version).Output()
		if err != nil {
			t.Fatalf("uv python find %s: %v", version, err)
		}
		probed, ok := Probe(t.Context(), command.NewOS(), strings.TrimSpace(string(path)))
		if !ok || len(probed.Stdlib) < 200 || !slices.Contains(probed.Keywords, "class") {
			t.Errorf("Python %s probe = %+v", version, probed)
		}
		if version == "3.10" && !slices.Contains(probed.Stdlib, "telnetlib") {
			t.Error("Python 3.10 ships telnetlib, but the probe did not list it")
		}
	}
}
