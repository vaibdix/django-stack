package scaffold

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vaibdix/django-cli/internal/config"
	"github.com/vaibdix/django-cli/internal/scaffold/command/commandtest"
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
		config := renderPreCommitConfig(preCommitData{MypyEnabled: true, UVAvailable: tc.uv}, tc.windows)
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
	if err := writeVSCodeConfig(root, cfg, NPM); err != nil {
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

func TestHooksRunTheProjectsOwnLockedTools(t *testing.T) {
	t.Parallel()

	config := renderPreCommitConfig(preCommitData{RuffEnabled: true, Templates: true, UVAvailable: true}, false)
	for _, want := range []string{"entry: uv run ruff check --fix", "entry: uv run ruff format", "entry: uv run djlint --lint --profile=django"} {
		if !strings.Contains(config, want) {
			t.Errorf("the hooks do not run the venv's tools; missing %q:\n%s", want, config)
		}
	}
	for _, remote := range []string{"astral-sh/ruff-pre-commit", "djlint/djLint"} {
		if strings.Contains(config, remote) {
			t.Errorf("%s is pulled as a remote hook, so its version drifts from the lockfile:\n%s", remote, config)
		}
	}

	apiOnly := renderPreCommitConfig(preCommitData{}, false)
	if strings.Contains(apiOnly, "djlint") {
		t.Errorf("a project without templates still runs djlint:\n%s", apiOnly)
	}
}

func TestPreCommitSetupNeverAutoupdatesHooks(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	fake := commandtest.NewFake()
	e := &Env{Cfg: config.Project{ProjectName: "demo", SetupPreCommit: true}, Runner: fake, Reporter: NopReporter{}}
	if err := e.applyPreCommit(root); err != nil {
		t.Fatalf("applyPreCommit: %v", err)
	}
	for _, line := range fake.CommandLines() {
		if strings.Contains(line, "autoupdate") {
			t.Errorf("generation rewrites hook versions from the network: %s", line)
		}
	}
}

func TestThePinnedHookRevisionsExistUpstream(t *testing.T) {
	if os.Getenv("DJANGO_STACK_NETWORK_TESTS") == "" {
		t.Skip("set DJANGO_STACK_NETWORK_TESTS=1 to check the pinned hook tags against GitHub")
	}
	for repo, tag := range map[string]string{
		"https://github.com/pre-commit/pre-commit-hooks": PreCommitHooksRev,
		"https://github.com/adamchainz/django-upgrade":   DjangoUpgradeRev,
	} {
		out, err := exec.CommandContext(t.Context(), "git", "ls-remote", "--tags", repo, "refs/tags/"+tag).Output()
		if err != nil || !strings.Contains(string(out), "refs/tags/"+tag) {
			t.Errorf("%s has no tag %s (err=%v); bump it in precommit.go", repo, tag, err)
		}
	}
}
