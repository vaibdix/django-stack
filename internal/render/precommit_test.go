package render

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/vaibdix/django-stack/internal/config"
	"github.com/vaibdix/django-stack/internal/livetest"
	"go.yaml.in/yaml/v3"
)

func TestTheMypyHookInstallsTheLockedVersionsWithOrWithoutUV(t *testing.T) {
	t.Parallel()

	for _, uv := range []bool{false, true} {
		config := RenderPreCommitConfig(PreCommitData{MypyEnabled: true, UVAvailable: uv, MypyDeps: []string{"django==6.1.2", "mypy==2.4.0"}})
		for _, want := range []string{"entry: mypy .", "language: python", "- django==6.1.2", "- mypy==2.4.0"} {
			if !strings.Contains(config, want) {
				t.Errorf("uv=%t: want %q in:\n%s", uv, want, config)
			}
		}
		if strings.Contains(config, "uv run") || strings.Contains(config, ".venv") || strings.Contains(config, "language: system\n        pass_filenames: false\n        types") {
			t.Errorf("uv=%t: a tool hook still depends on uv or a venv path:\n%s", uv, config)
		}
	}
}

func TestThePipPathConfigNamesNoInterpreterPath(t *testing.T) {
	t.Parallel()

	config := RenderPreCommitConfig(PreCommitData{
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
	pins, err := ExactPins([]string{"django-stubs[compatible-mypy]>=6.1,<6.2", "ruff"}, versions, true)
	if err != nil {
		t.Fatalf("exactPins: %v", err)
	}
	if strings.Join(pins, " ") != "django-stubs[compatible-mypy]==6.1.2 ruff==0.17.1" {
		t.Errorf("pins = %v", pins)
	}
	if _, err := ExactPins([]string{"mypy"}, versions, true); err == nil {
		t.Error("a package missing from the freeze was pinned")
	}
	if pins, _ := ExactPins([]string{"mypy"}, versions, false); pins != nil {
		t.Errorf("an unwanted hook got dependencies: %v", pins)
	}
}

func TestHooksRunTheProjectsOwnLockedTools(t *testing.T) {
	t.Parallel()

	config := RenderPreCommitConfig(PreCommitData{
		RuffEnabled: true, Templates: true, UVAvailable: true,
		RuffDeps: []string{"ruff==0.17.1"}, DjLintDeps: []string{"djlint==1.46.6"},
	})
	for _, want := range []string{"entry: ruff check --fix", "entry: ruff format", "entry: djlint --lint --profile=django", "- ruff==0.17.1", "- djlint==1.46.6", "entry: uv lock --check"} {
		if !strings.Contains(config, want) {
			t.Errorf("the hooks do not install the locked tools; missing %q:\n%s", want, config)
		}
	}
	for _, remote := range []string{"astral-sh/ruff-pre-commit", "djlint/djLint"} {
		if strings.Contains(config, remote) {
			t.Errorf("%s is pulled as a remote hook, so its version drifts from the lockfile:\n%s", remote, config)
		}
	}

	apiOnly := RenderPreCommitConfig(PreCommitData{})
	if strings.Contains(apiOnly, "djlint") {
		t.Errorf("a project without templates still runs djlint:\n%s", apiOnly)
	}
}

func TestThePinnedHookRevisionsExistUpstream(t *testing.T) {
	t.Parallel()

	livetest.Require(t, "check the pinned hook tags against GitHub")
	for repo, tag := range map[string]string{
		"https://github.com/pre-commit/pre-commit-hooks": preCommitHooksRev(),
		"https://github.com/adamchainz/django-upgrade":   djangoUpgradeRev(),
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
		data := PreCommitData{
			RuffEnabled: combination&1 != 0,
			MypyEnabled: combination&2 != 0,
			UVAvailable: combination&4 != 0,
			Templates:   combination&8 != 0,
			DjangoTrack: "6.1",
		}
		data.RuffDeps, data.DjLintDeps, data.MypyDeps = []string{"ruff==0.17.1"}, []string{"djlint==1.46.6"}, []string{"mypy==2.4.0"}
		rendered := RenderPreCommitConfig(data)
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

	e := &Project{Cfg: mustSpec(config.Choices{Name: "demo", AddOns: config.NewSet(config.PreCommit)})}
	for _, check := range e.VerificationChecks() {
		if strings.Join(check.Args, " ") == "-m pre_commit validate-config "+PreCommitConfigFile {
			return
		}
	}
	t.Errorf("a pre-commit project is not checked with validate-config: %+v", e.VerificationChecks())
}
