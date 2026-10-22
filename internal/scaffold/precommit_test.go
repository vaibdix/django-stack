package scaffold

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"

	"github.com/vaibdix/django-cli/internal/config"
	"github.com/vaibdix/django-cli/internal/scaffold/command/commandtest"
	"github.com/vaibdix/django-cli/internal/scaffold/pyenv"
)

func TestTheMypyHookChecksTheProjectWithOrWithoutUV(t *testing.T) {
	t.Parallel()

	withUV := renderPreCommitConfig(preCommitData{MypyEnabled: true, UVAvailable: true})
	if !strings.Contains(withUV, "entry: uv run mypy .") || !strings.Contains(withUV, "language: system") {
		t.Errorf("with uv the hook should run the project's own mypy:\n%s", withUV)
	}

	withPip := renderPreCommitConfig(preCommitData{MypyEnabled: true, MypyDeps: []string{"django==6.1.2", "mypy==2.4.0"}})
	for _, want := range []string{"entry: mypy .", "language: python", "- django==6.1.2", "- mypy==2.4.0"} {
		if !strings.Contains(withPip, want) {
			t.Errorf("without uv: want %q in:\n%s", want, withPip)
		}
	}
	if strings.Contains(withPip, "uv ") || strings.Contains(withPip, ".venv") {
		t.Errorf("without uv a hook still calls uv or a venv path:\n%s", withPip)
	}
}

func TestThePipPathConfigNamesNoInterpreterPath(t *testing.T) {
	t.Parallel()

	config := renderPreCommitConfig(preCommitData{
		RuffEnabled: true, MypyEnabled: true, Templates: true,
		RuffDeps: []string{"ruff==0.17.1"}, DjLintDeps: []string{"djlint==1.46.6"}, MypyDeps: []string{"mypy==2.4.0"},
	})
	for _, osBound := range []string{".venv", "Scripts", "python.exe", "bin/python"} {
		if strings.Contains(config, osBound) {
			t.Errorf("the config depends on the generating OS through %q:\n%s", osBound, config)
		}
	}
}

func TestExactPinsComeFromTheFreeze(t *testing.T) {
	t.Parallel()

	versions := map[string]string{"django-stubs": "6.1.2", "ruff": "0.17.1"}
	pins, err := exactPins([]string{"django-stubs[compatible-mypy]>=6.1,<6.2", "ruff"}, versions, true)
	if err != nil {
		t.Fatalf("exactPins: %v", err)
	}
	if strings.Join(pins, " ") != "django-stubs[compatible-mypy]==6.1.2 ruff==0.17.1" {
		t.Errorf("pins = %v", pins)
	}
	if _, err := exactPins([]string{"mypy"}, versions, true); err == nil {
		t.Error("a package missing from the freeze was pinned")
	}
	if pins, _ := exactPins([]string{"mypy"}, versions, false); pins != nil {
		t.Errorf("an unwanted hook got dependencies: %v", pins)
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

	config := renderPreCommitConfig(preCommitData{RuffEnabled: true, Templates: true, UVAvailable: true})
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

	apiOnly := renderPreCommitConfig(preCommitData{})
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

func TestEveryPreCommitConfigIsValid(t *testing.T) {
	t.Parallel()

	type hook struct {
		ID       string   `yaml:"id"`
		Entry    string   `yaml:"entry"`
		Language string   `yaml:"language"`
		Deps     []string `yaml:"additional_dependencies"`
	}
	type repo struct {
		Repo  string `yaml:"repo"`
		Hooks []hook `yaml:"hooks"`
	}
	for combination := range 16 {
		data := preCommitData{
			RuffEnabled: combination&1 != 0,
			MypyEnabled: combination&2 != 0,
			UVAvailable: combination&4 != 0,
			Templates:   combination&8 != 0,
			DjangoTrack: "6.1",
		}
		if !data.UVAvailable {
			data.RuffDeps, data.DjLintDeps, data.MypyDeps = []string{"ruff==0.17.1"}, []string{"djlint==1.46.6"}, []string{"mypy==2.4.0"}
		}
		rendered := renderPreCommitConfig(data)
		var parsed struct {
			Repos []repo `yaml:"repos"`
		}
		if err := yaml.Unmarshal([]byte(rendered), &parsed); err != nil {
			t.Fatalf("%+v: not YAML: %v\n%s", data, err, rendered)
		}
		if len(parsed.Repos) < 2 {
			t.Errorf("%+v: %d repos, want the two remote ones at least", data, len(parsed.Repos))
		}
		for _, r := range parsed.Repos {
			if len(r.Hooks) == 0 {
				t.Errorf("%+v: %s has no hooks, which pre-commit refuses:\n%s", data, r.Repo, rendered)
			}
			for _, h := range r.Hooks {
				if r.Repo == "local" && h.Language == "python" && len(h.Deps) == 0 {
					t.Errorf("%+v: the python hook %s has nothing to install:\n%s", data, h.ID, rendered)
				}
			}
		}
	}
}

func TestVerificationValidatesThePreCommitConfig(t *testing.T) {
	t.Parallel()

	e := &Env{Cfg: config.Project{ProjectName: "demo", SetupPreCommit: true}}
	for _, check := range e.verificationChecks() {
		if strings.Join(check.args, " ") == "-m pre_commit validate-config "+PreCommitConfigFile {
			return
		}
	}
	t.Errorf("a pre-commit project is not checked with validate-config: %+v", e.verificationChecks())
}

func TestAFailedPreCommitInstallFailsTheStep(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	fake := commandtest.NewFake()
	fake.FailWith(pyenv.InterpreterPath(root)+" -m pre_commit install", "not a git repository")
	e := &Env{Cfg: config.Project{ProjectName: "demo", SetupPreCommit: true, InitializeGit: true}, Runner: fake, Reporter: NopReporter{}}

	err := e.applyPreCommit(root)
	if err == nil || !strings.Contains(err.Error(), "pre-commit hooks") {
		t.Fatalf("applyPreCommit = %v, want the failed install reported", err)
	}
}
