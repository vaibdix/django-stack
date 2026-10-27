package scaffold

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/vaibdix/django-stack/internal/command/commandtest"
	"github.com/vaibdix/django-stack/internal/config"
	"github.com/vaibdix/django-stack/internal/port"
	"github.com/vaibdix/django-stack/internal/port/porttest"
	"github.com/vaibdix/django-stack/internal/pyenv"
	"github.com/vaibdix/django-stack/internal/render"
)

func writeReport(t *testing.T, root, group string, pins ...string) {
	t.Helper()
	install := make([]string, 0, len(pins))
	for _, pin := range pins {
		name, version, _ := strings.Cut(pin, "==")
		install = append(install, `{"metadata": {"name": "`+name+`", "version": "`+version+`"}}`)
	}
	venv := filepath.Join(root, pyenv.VenvDir)
	if err := os.MkdirAll(venv, 0o755); err != nil {
		t.Fatal(err)
	}
	report := `{"version": "1", "install": [` + strings.Join(install, ", ") + `]}`
	if err := os.WriteFile(filepath.Join(venv, "django-stack-"+group+"-report.json"), []byte(report), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestRequirementsKeepDevToolsOutOfTheRuntimeFile(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	fake := commandtest.NewFake()
	writeReport(t, root, "runtime", "asgiref==3.12.1", "Django==6.1.2", "sqlparse==0.6.0", "whitenoise==6.12.0")
	writeReport(t, root, "dev", "mypy==2.4.0", "mypy_extensions==1.1.0", "ruff==0.17.1")

	e := &Env{Cfg: mustSpec(config.Choices{Name: "demo"}), ports: port.Ports{Run: fake, Index: porttest.IndexFunc(fakeHashes)}}
	e.packages.AddRuntime("django==6.1.2", "whitenoise")
	e.packages.AddDev("ruff", "mypy")

	if err := e.at(root).writeRequirementsFiles(t.Context(), nil); err != nil {
		t.Fatalf("writeRequirementsFiles: %v", err)
	}

	runtime, _ := os.ReadFile(filepath.Join(root, render.RequirementsFile))
	dev, _ := os.ReadFile(filepath.Join(root, render.DevRequirementsFile))
	if !strings.Contains(string(runtime), "sqlparse==0.6.0 \\\n    --hash=sha256:sqlparse\n") {
		t.Errorf("requirements.txt does not pin the transitive dependencies:\n%s", runtime)
	}
	for _, tool := range []string{"ruff", "mypy"} {
		if strings.Contains(string(runtime), tool) {
			t.Errorf("requirements.txt, which production installs, carries %s:\n%s", tool, runtime)
		}
	}
	if !strings.HasPrefix(withoutComments(string(dev)), "-r requirements.txt\n") || !strings.Contains(string(dev), "ruff==0.17.1 \\\n    --hash=sha256:ruff") {
		t.Errorf("requirements-dev.txt does not layer the tools on the runtime set:\n%s", dev)
	}
}

func TestAnUnreadableDependencyTreeFailsTheRun(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	fake := commandtest.NewFake()
	fake.FailWithPrefix(pyenv.InterpreterPath(root)+" -m pip install --dry-run", "ResolutionImpossible")

	e := &Env{Cfg: mustSpec(config.Choices{Name: "demo"}), ports: port.Ports{Run: fake, Index: porttest.IndexFunc(fakeHashes)}}
	e.packages.AddRuntime("django==6.1.2")
	e.packages.AddDev("ruff")

	err := e.at(root).writeRequirementsFiles(t.Context(), nil)
	if err == nil || !strings.Contains(err.Error(), "dependency tree") {
		t.Fatalf("writeRequirementsFiles = %v, want the failed resolve reported", err)
	}
	if _, statErr := os.Stat(filepath.Join(root, render.RequirementsFile)); !os.IsNotExist(statErr) {
		t.Errorf("a partial %s was written after the resolve failed", render.RequirementsFile)
	}
}

func fakeHashes(_ context.Context, _, project, _ string) ([]string, error) {
	return []string{"sha256:" + project}, nil
}

func TestAFailedHashLookupFailsTheRunInsteadOfDroppingTheHashes(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	fake := commandtest.NewFake()
	writeReport(t, root, "runtime", "asgiref==3.12.1", "django==6.1.2")
	writeReport(t, root, "dev", "ruff==0.17.1")

	failing := func(_ context.Context, _, project, _ string) ([]string, error) {
		if project == "asgiref" {
			return nil, errors.New("PyPI unreachable")
		}
		return []string{"sha256:" + project}, nil
	}
	e := &Env{Cfg: mustSpec(config.Choices{Name: "demo"}), ports: port.Ports{Run: fake, Index: porttest.IndexFunc(failing)}}
	e.packages.AddRuntime("django==6.1.2")
	e.packages.AddDev("ruff")

	err := e.at(root).writeRequirementsFiles(t.Context(), nil)
	if err == nil || !strings.Contains(err.Error(), "PyPI unreachable") {
		t.Fatalf("writeRequirementsFiles = %v, want the failed lookup reported", err)
	}
	for _, name := range []string{render.RequirementsFile, render.DevRequirementsFile} {
		if _, statErr := os.Stat(filepath.Join(root, name)); !os.IsNotExist(statErr) {
			t.Errorf("%s was written without the hashes the run promised", name)
		}
	}
}

func TestTheClosureIsResolvedAgainstTheInstalledVersions(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	fake := commandtest.NewFake()
	writeReport(t, root, "runtime", "asgiref==3.12.1", "Django==6.1.2")
	e := &Env{Cfg: mustSpec(config.Choices{Name: "demo"}), ports: port.Ports{Run: fake}}

	closure, err := e.at(root).dependencyClosure(t.Context(), "runtime", []string{"django==6.1.2"}, map[string]string{"django": "6.1.2", "asgiref": "3.12.1"})
	if err != nil {
		t.Fatalf("dependencyClosure: %v", err)
	}
	if closure["django"] != "6.1.2" || closure["asgiref"] != "3.12.1" || len(closure) != 2 {
		t.Errorf("closure = %v, want django and asgiref at their installed versions", closure)
	}

	venv := filepath.Join(root, pyenv.VenvDir)
	want := pyenv.InterpreterPath(root) + " -m pip install --dry-run --ignore-installed --quiet --report " +
		filepath.Join(venv, "django-stack-runtime-report.json") + " --constraint " +
		filepath.Join(venv, "django-stack-constraints.txt") + " django==6.1.2"
	if !fake.Ran(want) {
		t.Errorf("pip was not asked for a constrained report; calls: %v", fake.CommandLines())
	}
	for _, leftover := range []string{"django-stack-runtime-report.json", "django-stack-constraints.txt"} {
		if _, err := os.Stat(filepath.Join(venv, leftover)); !os.IsNotExist(err) {
			t.Errorf("%s was left in the venv", leftover)
		}
	}
}

func TestAReportInAnUnknownFormatIsRefused(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	venv := filepath.Join(root, pyenv.VenvDir)
	if err := os.MkdirAll(venv, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(venv, "django-stack-runtime-report.json"), []byte(`{"version": "2", "install": []}`), 0o644); err != nil {
		t.Fatal(err)
	}
	e := &Env{Cfg: mustSpec(config.Choices{Name: "demo"}), ports: port.Ports{Run: commandtest.NewFake()}}

	if _, err := e.at(root).dependencyClosure(t.Context(), "runtime", []string{"django"}, nil); err == nil || !strings.Contains(err.Error(), "version") {
		t.Errorf("dependencyClosure = %v, want a version 2 report refused", err)
	}
}

func TestRequirementsSayWhichPlatformTheyWereResolvedFor(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeReport(t, root, "runtime", "django==6.1.2")
	e := &Env{
		Cfg:      mustSpec(config.Choices{Name: "demo"}),
		Resolved: config.Resolved{PythonVersion: "3.13.13"},
		ports:    port.Ports{Run: commandtest.NewFake(), Index: porttest.IndexFunc(fakeHashes)},
	}
	e.packages.AddRuntime("django==6.1.2")

	if err := e.at(root).writeRequirementsFiles(t.Context(), nil); err != nil {
		t.Fatalf("writeRequirementsFiles: %v", err)
	}
	content, _ := os.ReadFile(filepath.Join(root, render.RequirementsFile))
	if !strings.Contains(string(content), "Python 3.13.13 on "+goruntime.GOOS+"/"+goruntime.GOARCH) || !strings.Contains(string(content), "uv.lock") {
		t.Errorf("requirements.txt does not say what it was resolved for:\n%s", content)
	}
}

func withoutComments(content string) string {
	var kept []string
	for line := range strings.SplitSeq(content, "\n") {
		if !strings.HasPrefix(line, "#") {
			kept = append(kept, line)
		}
	}
	return strings.Join(kept, "\n")
}

func TestHashLookupsAreBoundedAndStopAtTheFirstFailure(t *testing.T) {
	t.Parallel()

	var mu sync.Mutex
	inFlight, peak, calls := 0, 0, 0
	lookup := func(ctx context.Context, _, project, _ string) ([]string, error) {
		mu.Lock()
		inFlight++
		calls++
		peak = max(peak, inFlight)
		mu.Unlock()
		defer func() { mu.Lock(); inFlight--; mu.Unlock() }()

		if project == "pkg-0" {
			return nil, errors.New("index unreachable")
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(20 * time.Millisecond):
			return []string{"sha256:" + project}, nil
		}
	}

	pins := map[string]string{}
	for i := range 40 {
		pins[fmt.Sprintf("pkg-%d", i)] = "1.0"
	}
	e := &Env{Cfg: mustSpec(config.Choices{Name: "demo"}), ports: port.Ports{Index: porttest.IndexFunc(lookup)}}

	_, err := e.requirementHashes(t.Context(), pins, map[string]string{"pkg-1": "1.0"})
	if err == nil || !strings.Contains(err.Error(), "pkg-0==1.0") {
		t.Fatalf("requirementHashes = %v, want the failing lookup named", err)
	}
	if peak > render.HashLookups {
		t.Errorf("%d lookups ran at once, want at most %d", peak, render.HashLookups)
	}
	if calls > 40 {
		t.Errorf("%d lookups for 40 distributions: one was looked up twice", calls)
	}
}
