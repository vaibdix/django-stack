package porttest

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"math/rand/v2"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/vaibdix/django-stack/internal/config"
	"github.com/vaibdix/django-stack/internal/report"
)

type memFile struct {
	data []byte
	mode fs.FileMode
	dir  bool
}

type MemFS struct {
	mu    sync.Mutex
	files map[string]memFile
}

func NewMemFS() *MemFS {
	return &MemFS{files: map[string]memFile{"/": {dir: true, mode: fs.ModeDir | 0o755}}}
}

func clean(name string) string { return path.Clean("/" + filepath.ToSlash(name)) }

func (m *MemFS) Stat(name string) (fs.FileInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	f, ok := m.files[clean(name)]
	if !ok {
		return nil, &fs.PathError{Op: "stat", Path: name, Err: fs.ErrNotExist}
	}
	return info{name: path.Base(clean(name)), file: f}, nil
}

func (m *MemFS) ReadDir(name string) ([]fs.DirEntry, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	dir := clean(name)
	if f, ok := m.files[dir]; !ok || !f.dir {
		return nil, &fs.PathError{Op: "readdir", Path: name, Err: fs.ErrNotExist}
	}
	var entries []fs.DirEntry
	for p, f := range m.files {
		if p != dir && path.Dir(p) == dir {
			entries = append(entries, fs.FileInfoToDirEntry(info{name: path.Base(p), file: f}))
		}
	}
	slices.SortFunc(entries, func(a, b fs.DirEntry) int { return strings.Compare(a.Name(), b.Name()) })
	return entries, nil
}

func (m *MemFS) ReadFile(name string) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	f, ok := m.files[clean(name)]
	if !ok || f.dir {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
	}
	return slices.Clone(f.data), nil
}

func (m *MemFS) MkdirAll(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.mkdirAll(clean(name))
}

func (m *MemFS) mkdirAll(dir string) error {
	for p := dir; ; p = path.Dir(p) {
		if f, ok := m.files[p]; ok && !f.dir {
			return &fs.PathError{Op: "mkdir", Path: p, Err: fs.ErrExist}
		}
		if p == "/" {
			break
		}
	}
	for p := dir; p != "/"; p = path.Dir(p) {
		m.files[p] = memFile{dir: true, mode: fs.ModeDir | 0o755}
	}
	return nil
}

func (m *MemFS) RemoveAll(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	target := clean(name)
	for p := range m.files {
		if p == target || strings.HasPrefix(p, target+"/") {
			delete(m.files, p)
		}
	}
	return nil
}

func (m *MemFS) WriteIn(root, rel string, data []byte, mode fs.FileMode) error {
	if !filepath.IsLocal(rel) {
		return &fs.PathError{Op: "write", Path: rel, Err: errors.New("path escapes from parent")}
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	base := clean(root)
	if f, ok := m.files[base]; !ok || !f.dir {
		return fmt.Errorf("open the project directory: %w", &fs.PathError{Op: "openat", Path: root, Err: fs.ErrNotExist})
	}
	target := path.Join(base, filepath.ToSlash(rel))
	if err := m.mkdirAll(path.Dir(target)); err != nil {
		return err
	}
	m.files[target] = memFile{data: slices.Clone(data), mode: mode}
	return nil
}

func (m *MemFS) Files(root string) map[string]string {
	m.mu.Lock()
	defer m.mu.Unlock()
	base := clean(root)
	tree := map[string]string{}
	for p, f := range m.files {
		if !f.dir && strings.HasPrefix(p, base+"/") {
			tree[strings.TrimPrefix(p, base+"/")] = fmt.Sprintf("%v\n%s", f.mode, f.data)
		}
	}
	return tree
}

type info struct {
	name string
	file memFile
}

func (i info) Name() string { return i.name }

func (i info) Size() int64 { return int64(len(i.file.data)) }

func (i info) Mode() fs.FileMode { return i.file.mode }

func (i info) ModTime() time.Time { return time.Time{} }

func (i info) IsDir() bool { return i.file.dir }

func (i info) Sys() any { return nil }

type Index map[string][]string

func (x Index) FileHashes(ctx context.Context, index, project, version string) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	hashes, ok := x[project+"=="+version]
	if !ok {
		return nil, fmt.Errorf("%s %s is not on the index", project, version)
	}
	return slices.Clone(hashes), nil
}

type Docker struct {
	mu    sync.Mutex
	Calls []string
	Err   map[string]error
}

func (d *Docker) record(call string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.Calls = append(d.Calls, call)
	return d.Err[strings.Fields(call)[0]]
}

func (d *Docker) Ready(ctx context.Context, project string, db config.Database) error {
	return d.record("ready " + project + " " + string(db))
}

func (d *Docker) Start(ctx context.Context, notes report.Steps, dir, project string, db config.Database) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return d.record("start " + project + " " + string(db))
}

func (d *Docker) Stop(ctx context.Context, dir string) error {
	return d.record("stop")
}

func (d *Docker) FreePort(preferred int) (string, error) {
	return strconv.Itoa(preferred), d.record("port")
}

func Clock(at time.Time) func() time.Time { return func() time.Time { return at } }

type seeded struct {
	mu     sync.Mutex
	stream *rand.ChaCha8
}

func (s *seeded) Read(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.stream.Read(p)
}

func Rand(seed uint64) io.Reader {
	var key [32]byte
	binary.LittleEndian.PutUint64(key[:], seed)
	return &seeded{stream: rand.NewChaCha8(key)}
}

type IndexFunc func(ctx context.Context, index, project, version string) ([]string, error)

func (f IndexFunc) FileHashes(ctx context.Context, index, project, version string) ([]string, error) {
	return f(ctx, index, project, version)
}
