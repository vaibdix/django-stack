package adapter_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/vaibdix/django-stack/internal/adapter"
	"github.com/vaibdix/django-stack/internal/command"
	"github.com/vaibdix/django-stack/internal/command/commandtest"
	"github.com/vaibdix/django-stack/internal/config"
	"github.com/vaibdix/django-stack/internal/dbdocker"
	"github.com/vaibdix/django-stack/internal/event"
	"github.com/vaibdix/django-stack/internal/livetest"
	"github.com/vaibdix/django-stack/internal/port"
	"github.com/vaibdix/django-stack/internal/port/porttest"
)

func TestFileSystemsKeepTheSameContract(t *testing.T) {
	t.Parallel()

	mem := porttest.NewMemFS()
	if err := mem.MkdirAll("/work/shop"); err != nil {
		t.Fatal(err)
	}
	for name, c := range map[string]struct {
		fsys port.FS
		root string
	}{
		"os":     {adapter.OSFS{}, t.TempDir()},
		"memory": {mem, "/work/shop"},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			fileSystemContract(t, c.fsys, c.root)
		})
	}
}

func fileSystemContract(t *testing.T, fsys port.FS, root string) {
	t.Helper()

	if err := fsys.WriteIn(root, filepath.Join("shop", "settings.py"), []byte("DEBUG = True\n"), 0o600); err != nil {
		t.Fatalf("WriteIn: %v", err)
	}
	got, err := fsys.ReadFile(filepath.Join(root, "shop", "settings.py"))
	if err != nil || string(got) != "DEBUG = True\n" {
		t.Errorf("ReadFile = %q, %v", got, err)
	}
	info, err := fsys.Stat(filepath.Join(root, "shop", "settings.py"))
	if err != nil || (runtime.GOOS != "windows" && info.Mode().Perm() != 0o600) || info.IsDir() {
		t.Errorf("Stat = %v, %v; want a 0600 file", info, err)
	}
	if err := fsys.WriteIn(root, filepath.Join("..", "escaped.txt"), []byte("x"), 0o644); err == nil {
		t.Error("a write outside the project folder was allowed")
	}
	if err := fsys.WriteIn(filepath.Join(root, "missing"), "a.txt", []byte("x"), 0o644); err == nil {
		t.Error("a write into a folder that does not exist was allowed")
	}
	if err := fsys.MkdirAll(filepath.Join(root, "b", "c")); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	entries, err := fsys.ReadDir(root)
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	if err != nil || !slices.Equal(names, []string{"b", "shop"}) {
		t.Errorf("ReadDir = %v, %v; want [b shop] in name order", names, err)
	}
	if err := fsys.RemoveAll(filepath.Join(root, "shop")); err != nil {
		t.Fatalf("RemoveAll: %v", err)
	}
	if _, err := fsys.Stat(filepath.Join(root, "shop", "settings.py")); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("Stat after RemoveAll = %v, want fs.ErrNotExist", err)
	}
	if err := fsys.RemoveAll(filepath.Join(root, "never-made")); err != nil {
		t.Errorf("RemoveAll of a missing path = %v, want nil", err)
	}
}

func TestPackageIndexesKeepTheSameContract(t *testing.T) {
	t.Parallel()

	page := `{"meta":{"api-version":"1.1"},"name":"demo","files":[
		{"filename":"demo-1.0-py3-none-any.whl","hashes":{"sha256":"bb"}},
		{"filename":"demo-1.0.tar.gz","hashes":{"sha256":"aa"}},
		{"filename":"demo-2.0.tar.gz","hashes":{"sha256":"cc"}}]}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/simple/demo/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/vnd.pypi.simple.v1+json")
		_, _ = io.WriteString(w, page)
	}))
	t.Cleanup(server.Close)

	for name, index := range map[string]port.Index{
		"http":   adapter.PyPI{Client: server.Client()},
		"memory": porttest.Index{"demo==1.0": {"sha256:aa", "sha256:bb"}, "demo==2.0": {"sha256:cc"}},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			hashes, err := index.FileHashes(t.Context(), server.URL+"/simple/", "demo", "1.0")
			if err != nil || !slices.Equal(hashes, []string{"sha256:aa", "sha256:bb"}) {
				t.Errorf("FileHashes(demo 1.0) = %v, %v; want both files' hashes, sorted", hashes, err)
			}
			if _, err := index.FileHashes(t.Context(), server.URL+"/simple/", "demo", "9.9"); err == nil {
				t.Error("a version the index does not list returned hashes")
			}
			cancelled, cancel := context.WithCancel(t.Context())
			cancel()
			if _, err := index.FileHashes(cancelled, server.URL+"/simple/", "demo", "1.0"); err == nil {
				t.Error("a cancelled lookup succeeded")
			}
		})
	}
}

func TestRunnersKeepTheSameContract(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("uses echo from a POSIX shell")
	}

	fake := commandtest.NewFake()
	fake.Responses["echo hi"] = []byte("hi\n")
	for name, run := range map[string]port.Runner{"os": command.NewOS(), "fake": fake} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			out, err := run.Run(t.Context(), "", "echo", "hi")
			if err != nil || string(out) != "hi\n" {
				t.Errorf("Run(echo hi) = %q, %v", out, err)
			}
			cancelled, cancel := context.WithCancel(t.Context())
			cancel()
			_, err = run.Run(cancelled, "", "echo", "hi")
			if _, ok := errors.AsType[*command.Error](err); !ok {
				t.Errorf("a cancelled Run returned %v, want a *command.Error", err)
			}
		})
	}
}

func TestDockerAdaptersKeepTheSameContract(t *testing.T) {
	t.Parallel()

	adapters := map[string]port.Docker{"fake": &porttest.Docker{}}
	if livetest.Enabled() {
		adapters["cli"] = adapter.DockerCLI{Run: command.NewOS()}
	}
	for name, docker := range adapters {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			cancelled, cancel := context.WithCancel(t.Context())
			cancel()
			if err := docker.Start(cancelled, event.Discard{}, t.TempDir(), "contract", config.PostgresDocker); err == nil {
				t.Error("Start with a cancelled context succeeded")
			}
			if name == "cli" {
				dockerLifecycle(t, docker)
			}
		})
	}
}

func dockerLifecycle(t *testing.T, docker port.Docker) {
	t.Helper()
	project := fmt.Sprintf("contract-%d", time.Now().UnixNano())
	dir := t.TempDir()
	if err := dbdocker.WriteComposeFile(dir, project, config.PostgresDocker); err != nil {
		t.Fatal(err)
	}
	port, err := dbdocker.AvailablePort(55442)
	if err != nil {
		t.Fatal(err)
	}
	env := "DB_NAME=contract\nDB_USER=django_user\nDB_PASSWORD=contractpw\nDB_ROOT_PASSWORD=contractroot\nDB_PORT=" + port + "\n"
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte(env), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := docker.Ready(t.Context(), project, config.PostgresDocker); err != nil {
		livetest.MissingTool(t, "docker is not ready: %v", err)
	}
	t.Cleanup(func() { _ = docker.Stop(context.WithoutCancel(t.Context()), dir) })
	if err := docker.Start(t.Context(), event.Discard{}, dir, project, config.PostgresDocker); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if err := docker.Stop(t.Context(), dir); err != nil {
		t.Errorf("Stop: %v", err)
	}
}

func TestRandomnessSourcesFillWhatTheyAreAskedFor(t *testing.T) {
	t.Parallel()

	for name, source := range map[string]io.Reader{
		"crypto": adapter.Complete(port.Ports{}).Rand,
		"seeded": porttest.Rand(7),
	} {
		first, second := make([]byte, 38), make([]byte, 38)
		if _, err := io.ReadFull(source, first); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if _, err := io.ReadFull(source, second); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if bytes.Equal(first, second) {
			t.Errorf("%s returned the same bytes twice", name)
		}
	}
	a, b := make([]byte, 16), make([]byte, 16)
	_, _ = io.ReadFull(porttest.Rand(7), a)
	_, _ = io.ReadFull(porttest.Rand(7), b)
	if !bytes.Equal(a, b) {
		t.Error("the same seed gave different bytes")
	}
}

func TestFreePortsComeBackAsNumbers(t *testing.T) {
	t.Parallel()

	for name, docker := range map[string]port.Docker{"cli": adapter.DockerCLI{}, "fake": &porttest.Docker{}} {
		got, err := docker.FreePort(55462)
		if err != nil {
			t.Fatalf("%s: FreePort: %v", name, err)
		}
		if n, err := strconv.Atoi(got); err != nil || n < 55462 || n >= 55562 {
			t.Errorf("%s: FreePort(55462) = %q, want a port at or just above the preferred one", name, got)
		}
	}
}
