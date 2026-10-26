package pyenv

import (
	"io/fs"
	"os/exec"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/vaibdix/django-stack/internal/command/commandtest"
	"github.com/vaibdix/django-stack/internal/report"
)

type fakeFile struct{ dir bool }

func (f fakeFile) Name() string       { return "python3" }
func (f fakeFile) Size() int64        { return 0 }
func (f fakeFile) Mode() fs.FileMode  { return 0o755 }
func (f fakeFile) ModTime() time.Time { return time.Time{} }
func (f fakeFile) IsDir() bool        { return f.dir }
func (f fakeFile) Sys() any           { return nil }

type fakeMachine struct {
	goos     string
	path     map[string]string
	files    map[string]bool
	env      map[string]string
	versions map[string]string
}

func (m fakeMachine) host() Host {
	return Host{
		GOOS: m.goos,
		LookPath: func(name string) (string, error) {
			if p, ok := m.path[name]; ok {
				return p, nil
			}
			return "", exec.ErrNotFound
		},
		Stat: func(name string) (fs.FileInfo, error) {
			if dir, ok := m.files[name]; ok {
				return fakeFile{dir: dir}, nil
			}
			return nil, fs.ErrNotExist
		},
		Getenv: func(name string) string { return m.env[name] },
	}
}

func (m fakeMachine) runner() *commandtest.Fake {
	fake := commandtest.NewFake()
	for path, version := range m.versions {
		fake.Responses[path+" -c "+probeScript] = []byte(`{"version": "` + version + `", "stdlib": ["os"], "keywords": ["if"]}`)
	}
	return fake
}

func TestDiscoverCombinesUVThePathAndTheKnownLocations(t *testing.T) {
	t.Parallel()

	m := fakeMachine{
		goos: "linux",
		path: map[string]string{"uv": "/bin/uv", "python3": "/usr/bin/python3"},
		files: map[string]bool{
			"/usr/bin/python3":       false,
			"/opt/uv/python3.13":     false,
			"/usr/local/bin/python3": false,
			"/usr/local/bin/python":  true,
		},
		versions: map[string]string{"/usr/bin/python3": "3.12.4", "/opt/uv/python3.13": "3.13.2", "/usr/local/bin/python3": "3.9.1"},
	}
	fake := m.runner()
	fake.Responses["uv python list --only-installed --output-format json"] = []byte(`[
		{"path": "/opt/uv/python3.13", "implementation": "cpython"},
		{"path": "/opt/uv/pypy", "implementation": "pypy"},
		{"path": "/opt/uv/free", "implementation": "cpython", "variant": "freethreaded"}
	]`)
	fake.Responses["which -a python3"] = []byte("/usr/bin/python3\n")
	fake.Errors["which -a python"] = &exec.ExitError{}

	options, err := m.host().Discover(t.Context(), fake)
	if err == nil {
		t.Log("PATH search reported no error")
	}
	var paths []string
	for _, option := range options {
		paths = append(paths, option.Path)
	}
	if !slices.Equal(paths, []string{"/opt/uv/python3.13", "/usr/bin/python3"}) {
		t.Fatalf("discovered %v, want uv's 3.13 first and the PATH 3.12, without 3.9, pypy, a free-threaded build or a directory", paths)
	}
	if !options[1].IsDefault || !strings.HasSuffix(options[1].Label, "[shell default]") {
		t.Errorf("the python3 on PATH is the shell default: %+v", options[1])
	}
}

func TestDiscoverReportsAnUnreadableUVListing(t *testing.T) {
	t.Parallel()

	m := fakeMachine{goos: "darwin", path: map[string]string{"uv": "/bin/uv"}}
	fake := m.runner()
	fake.Responses["uv python list --only-installed --output-format json"] = []byte("not json")
	if _, err := m.host().Discover(t.Context(), fake); err == nil || !strings.Contains(err.Error(), "JSON") {
		t.Errorf("Discover = %v, want the unreadable uv listing reported", err)
	}

	failing := m.runner()
	failing.FailWith("uv python list --only-installed --output-format json", "boom")
	if _, err := m.host().Discover(t.Context(), failing); err == nil || !strings.Contains(err.Error(), "uv python list failed") {
		t.Errorf("Discover = %v, want the failed uv listing reported", err)
	}
}

func TestWithoutUVDiscoveryAsksNothingOfIt(t *testing.T) {
	t.Parallel()

	m := fakeMachine{goos: "linux"}
	fake := m.runner()
	if _, err := m.host().uvManagedPythonPaths(t.Context(), fake); err != nil {
		t.Fatalf("uvManagedPythonPaths without uv = %v", err)
	}
	if len(fake.Calls()) != 0 {
		t.Errorf("uv is not installed, yet discovery ran %v", fake.CommandLines())
	}
}

func TestKnownLocationsFollowTheOperatingSystem(t *testing.T) {
	t.Parallel()

	darwin := fakeMachine{goos: "darwin"}.host().knownPythonPaths()
	if !slices.Contains(darwin, "/opt/homebrew/bin/python3") {
		t.Errorf("macOS candidates miss Homebrew: %v", darwin)
	}
	windows := fakeMachine{goos: Windows, env: map[string]string{"LOCALAPPDATA": `C:\Users\u\AppData\Local`}}.host().knownPythonPaths()
	if len(windows) == 0 || !strings.Contains(windows[0], "Python3") {
		t.Errorf("Windows candidates come from LOCALAPPDATA: %v", windows)
	}
}

func TestNormalisingRejectsWhatCannotRun(t *testing.T) {
	t.Parallel()

	h := fakeMachine{
		goos:  "linux",
		path:  map[string]string{"python3": "/usr/bin/python3"},
		files: map[string]bool{"/usr/bin/python3": false, "/usr/lib": true},
	}.host()

	for path, want := range map[string]bool{
		"":                       false,
		"python3":                true,
		"python9":                false,
		"/usr/lib":               false,
		"/missing/python3":       false,
		"/proj/.venv/bin/python": false,
		"/usr/bin/python3":       true,
	} {
		if _, ok := h.NormalizeExecutablePath(path); ok != want {
			t.Errorf("NormalizeExecutablePath(%q) ok = %v, want %v", path, ok, want)
		}
	}
}

func TestCommandWorksNeedsTheBinaryAndAVersion(t *testing.T) {
	t.Parallel()

	h := fakeMachine{path: map[string]string{"git": "/usr/bin/git", "silent": "/usr/bin/silent"}}.host()
	fake := commandtest.NewFake()
	fake.Responses["git --version"] = []byte("git version 2.47.0\n")

	if !h.CommandWorks(t.Context(), fake, "git") {
		t.Error("git is on PATH and answers --version")
	}
	if h.CommandWorks(t.Context(), fake, "silent") {
		t.Error("a binary that prints no version is not taken as working")
	}
	if h.CommandWorks(t.Context(), fake, "node") {
		t.Error("a binary that is not on PATH is not taken as working")
	}
	if fake.Ran("node --version") {
		t.Error("a missing binary was run anyway")
	}
}

func TestInstallAllAndProbeVersionGoThroughTheRunner(t *testing.T) {
	t.Parallel()

	fake := commandtest.NewFake()
	fake.Responses["uv --version"] = []byte("uv 0.9.4\n")
	if version, err := ProbeVersion(t.Context(), fake, "uv"); err != nil || !strings.Contains(version, "0.9.4") {
		t.Errorf("ProbeVersion = %q, %v", version, err)
	}

	if err := InstallAll(t.Context(), fake, report.Nop{}, UV, "/proj", "django==6.1.2"); err != nil {
		t.Fatalf("InstallAll: %v", err)
	}
	name, args := PackageInstallArgs(UV, "/proj", "django==6.1.2")
	if !fake.Ran(name + " " + strings.Join(args, " ")) {
		t.Errorf("InstallAll ran %v, want %s %v", fake.CommandLines(), name, args)
	}

	fake.FailWithPrefix(name, "network down")
	if err := InstallAll(t.Context(), fake, report.Nop{}, UV, "/proj", "django"); err == nil || !strings.Contains(err.Error(), "install the project") {
		t.Errorf("InstallAll = %v, want the failed install reported", err)
	}
}

func TestFreezeCommandFollowsTheToolchain(t *testing.T) {
	t.Parallel()

	if name, args := FreezeCommand(UV, "/proj"); name != "uv" || !slices.Contains(args, "freeze") {
		t.Errorf("uv freeze = %s %v", name, args)
	}
	if name, args := FreezeCommand(Pip, "/proj"); !strings.Contains(name, VenvDir) || !slices.Equal(args, []string{"-m", "pip", "freeze"}) {
		t.Errorf("pip freeze = %s %v", name, args)
	}
}
