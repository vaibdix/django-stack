package scaffold

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"math/rand/v2"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/vaibdix/django-stack/internal/command/commandtest"
	"github.com/vaibdix/django-stack/internal/config"
	"github.com/vaibdix/django-stack/internal/port"
	"github.com/vaibdix/django-stack/internal/pyenv"
	"github.com/vaibdix/django-stack/internal/render"
)

type installingRunner struct {
	*commandtest.Fake
	mu        sync.Mutex
	installed []string
}

func (r *installingRunner) Run(ctx context.Context, dir, name string, args ...string) ([]byte, error) {
	switch {
	case name == "python3" && len(args) == 2 && args[0] == "-c":
		return []byte(`{"version": "3.13.2", "stdlib": ["os", "sys"], "keywords": ["class"]}`), nil
	case name == "uv" && len(args) > 2 && args[0] == "pip" && args[1] == "install":
		r.record(args[2:])
	case name == "uv" && len(args) > 1 && args[0] == "pip" && args[1] == "freeze":
		return r.freeze(), nil
	}
	return r.Fake.Run(ctx, dir, name, args...)
}

func (r *installingRunner) record(args []string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := 0; i < len(args); i++ {
		switch {
		case args[i] == "--python" || args[i] == "--exclude-newer":
			i++
		case !strings.HasPrefix(args[i], "-"):
			r.installed = append(r.installed, args[i])
		}
	}
}

func (r *installingRunner) freeze() []byte {
	r.mu.Lock()
	defer r.mu.Unlock()
	var b strings.Builder
	for _, spec := range r.installed {
		name, _, _ := strings.Cut(spec, "[")
		name = name[:strings.IndexFunc(name+"=", func(c rune) bool { return strings.ContainsRune("=<>!~;", c) })]
		version := "1.0.0"
		for _, op := range []string{"==", ">="} {
			if _, after, ok := strings.Cut(spec, op); ok {
				version, _, _ = strings.Cut(after, ",")
				break
			}
		}
		fmt.Fprintf(&b, "%s==%s\n", name, version)
	}
	return []byte(b.String())
}

func generateWithFixedPorts(t *testing.T, cfg config.Choices, seed byte) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), cfg.Name)
	e := New(specOf(t, cfg), config.Resolved{PythonPath: "python3", PythonVersion: "Python 3.13.2"}, Options{
		Root:        root,
		Ports:       port.Ports{Run: &installingRunner{Fake: commandtest.NewFake()}, Clock: reviewClock, Rand: rand.NewChaCha8([32]byte{seed})},
		Toolchain:   pyenv.UV,
		NodeManager: render.Pnpm,
	})
	if err := e.Generate(t.Context()); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	return root
}

func readTree(t *testing.T, root string) map[string][]byte {
	t.Helper()
	files := map[string][]byte{}
	err := fs.WalkDir(os.DirFS(root), ".", func(name string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		content, err := fs.ReadFile(os.DirFS(root), name)
		files[name] = content
		return err
	})
	if err != nil {
		t.Fatalf("reading %s: %v", root, err)
	}
	return files
}

func TestTheSameSpecClockAndRandGenerateTheSameProject(t *testing.T) {
	t.Parallel()

	cfg := config.Choices{
		Name:     "shop",
		App:      "catalog",
		Django:   "6.1.2",
		Database: config.SQLite,
		Stack:    config.StackTailwindHTMX,
		AddOns:   config.NewSet(config.RestFramework, config.AllAuth, config.Ruff, config.Mypy, config.Pydantic, config.MultiEnv),
	}
	first, second := readTree(t, generateWithFixedPorts(t, cfg, 7)), readTree(t, generateWithFixedPorts(t, cfg, 7))
	if len(first) == 0 {
		t.Fatal("the run wrote no files")
	}
	for name, content := range first {
		if again, ok := second[name]; !ok {
			t.Errorf("%s was written by the first run only", name)
		} else if !bytes.Equal(content, again) {
			t.Errorf("%s differs between two runs with the same spec, clock and randomness", name)
		}
	}
	for name := range second {
		if _, ok := first[name]; !ok {
			t.Errorf("%s was written by the second run only", name)
		}
	}
}

func TestTheRandPortIsTheOnlySourceOfTheSecret(t *testing.T) {
	t.Parallel()

	cfg := config.Choices{Name: "shop", App: "catalog", Django: "6.1.2", Database: config.SQLite, Stack: config.StackStandard}
	first, second := readTree(t, generateWithFixedPorts(t, cfg, 7)), readTree(t, generateWithFixedPorts(t, cfg, 8))
	if bytes.Equal(first[render.EnvFile], second[render.EnvFile]) {
		t.Errorf("two seeds wrote the same %s, so the secret does not come from the Rand port", render.EnvFile)
	}
	for name, content := range first {
		if name != render.EnvFile && !bytes.Equal(content, second[name]) {
			t.Errorf("%s depends on the randomness; only the secret should", name)
		}
	}
}
