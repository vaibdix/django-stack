package scaffold

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vaibdix/django-cli/internal/config"
)

func TestDjangoTrackDerivesTheSeriesFromTheSelectedVersion(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"6.1":    "6.1",
		"6.1.2":  "6.1",
		"6.0.3":  "6.0",
		"5.2.12": "5.2",
		"4.2":    "4.2",
	}

	for version, want := range cases {
		e := &Env{Cfg: config.Project{DjangoVersion: version}}
		if got := e.djangoTrack(); got != want {
			t.Errorf("djangoTrack() for %q = %q, want %q", version, got, want)
		}
	}
}

func TestDefaultDjangoTrackIsNotStale(t *testing.T) {
	t.Parallel()

	e := &Env{Cfg: config.Project{}}
	got := e.djangoTrack()

	if got != defaultDjangoTrack {
		t.Fatalf("an unset version should fall back to %q, got %q", defaultDjangoTrack, got)
	}

	config := e.generatePreCommitConfig()
	if !strings.Contains(config, `--target-version, "`+defaultDjangoTrack+`"`) {
		t.Errorf("the generated pre-commit config does not target %q:\n%s", defaultDjangoTrack, config)
	}

	if defaultDjangoTrack < "6.1" {
		t.Errorf("defaultDjangoTrack is %q; django-upgrade will decline every fix newer than that, "+
			"which is what four releases of drift looked like last time", defaultDjangoTrack)
	}
}

func TestTheMypyHookChecksTheProjectWithOrWithoutUV(t *testing.T) {
	t.Parallel()

	cases := []struct {
		uv, windows bool
		entry       string
	}{
		{uv: true, entry: "entry: uv run mypy ."},
		{uv: false, entry: "entry: .venv/bin/python -m mypy ."},
		{uv: false, windows: true, entry: "entry: .venv/Scripts/python.exe -m mypy ."},
	}
	for _, tc := range cases {
		config := renderPreCommitConfig(preCommitData{MypyEnabled: true, UVAvailable: tc.uv, DjLintVersion: DjLintVersion}, tc.windows)
		if !strings.Contains(config, tc.entry) {
			t.Errorf("uv=%v windows=%v: want %q in:\n%s", tc.uv, tc.windows, tc.entry, config)
		}
		if !tc.uv && strings.Contains(config, "uv ") {
			t.Errorf("without uv installed, a hook still calls uv:\n%s", config)
		}
	}
}

func TestEveryGeneratedJSONFileEndsWithANewline(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	cfg := config.Project{ProjectName: "demo", AppName: "catalog", SetupTailwind: true, CreateTemplates: true}
	if err := writeVSCodeConfig(root, cfg, npmManager); err != nil {
		t.Fatalf("writeVSCodeConfig: %v", err)
	}
	e := &Env{Cfg: cfg, Reporter: NopReporter{}}
	if err := e.ensurePackageJSON(root); err != nil {
		t.Fatalf("ensurePackageJSON: %v", err)
	}
	if err := e.updatePackageJSONForTailwind(root); err != nil {
		t.Fatalf("updatePackageJSONForTailwind: %v", err)
	}

	for _, name := range []string{".vscode/tasks.json", ".vscode/launch.json", ".vscode/extensions.json", "package.json"} {
		data, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			t.Fatalf("reading %s: %v", name, err)
		}
		if !strings.HasSuffix(string(data), "}\n") {
			t.Errorf("%s does not end with a newline, so end-of-file-fixer rewrites it on the first commit", name)
		}
	}
}
