package plan

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/vaibdix/django-stack/internal/config"
	"github.com/vaibdix/django-stack/internal/render"
)

func specFor(t *testing.T, cfg config.Choices) config.Spec {
	t.Helper()
	if cfg.Name == "" {
		cfg.Name = "demo"
	}
	if cfg.Django == "" {
		cfg.Django = "6.1.2"
	}
	if cfg.Database == "" {
		cfg.Database = config.SQLite
	}
	if cfg.Stack == "" {
		cfg.Stack = config.StackAPIOnly
	}
	if cfg.App == "" && cfg.NeedsAppName() {
		cfg.App = "catalog"
	}
	spec, _, err := config.Parse(cfg, config.Interpreter{}, time.Date(2026, 10, 27, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("config.Parse(%+v): %v", cfg, err)
	}
	return spec
}

func everything(t *testing.T) config.Spec {
	t.Helper()
	return specFor(t, config.Choices{
		App:      "catalog",
		Database: config.MySQLDocker,
		Stack:    config.StackTailwindHTMX,
		AddOns:   config.NewSet(config.Git, config.RestFramework, config.AllAuth, config.Ruff, config.Mypy, config.PreCommit, config.MultiEnv),
	})
}

func dependsOn(after, before string) bool {
	needs := map[string][]string{}
	for _, e := range Effects() {
		needs[e.ID] = e.Needs
	}
	seen := map[string]bool{}
	queue := []string{after}
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		for _, need := range needs[id] {
			if need == before {
				return true
			}
			if !seen[need] {
				seen[need] = true
				queue = append(queue, need)
			}
		}
	}
	return false
}

func TestEveryNeedIsAnEarlierEffect(t *testing.T) {
	t.Parallel()

	seen := map[string]bool{}
	for _, e := range Effects() {
		if e.ID == "" || e.Label == "" || seen[e.ID] {
			t.Errorf("effect %q is unnamed, unlabelled or listed twice", e.ID)
		}
		for _, need := range e.Needs {
			if !seen[need] {
				t.Errorf("%q needs %q, which is not declared before it", e.ID, need)
			}
		}
		if e.Timeout <= 0 {
			t.Errorf("%q has no timeout", e.ID)
		}
		seen[e.ID] = true
	}
}

func TestLoadBearingOrderingsHold(t *testing.T) {
	t.Parallel()

	constraints := []struct {
		before, after, why string
	}{
		{WriteEnv, StartDatabase, "docker compose reads the credentials and port from .env"},
		{WriteCompose, StartDatabase, "the database is started from the compose file the project keeps"},
		{StartDatabase, Migrate, "migrations need the database to be healthy"},
		{WriteEnv, Migrate, "the settings read the database and secret key from .env"},
		{WriteSharedApps, Migrate, "AUTH_USER_MODEL must be set before the first migration is applied"},
		{WriteApp, Migrate, "the app's migrations must exist before they are applied"},
		{SetupTailwind, BuildCSS, "the build runs the Tailwind CLI that the setup installs"},
		{WriteTemplates, BuildCSS, "the stylesheet must be compiled after the templates it scans"},
		{WriteAllAuth, BuildCSS, "the allauth pages must exist before the stylesheet is compiled"},
		{WriteApp, BuildCSS, "the app templates must exist before the stylesheet is compiled"},
		{WriteDRF, BuildCSS, "the REST example templates must exist before the stylesheet is compiled"},
		{InstallPackages, LockDependencies, "the manifest records what was installed, so every installer must run first"},
		{WriteDRF, LockDependencies, "the manifest records what was installed, so every installer must run first"},
		{WriteAllAuth, LockDependencies, "the manifest records what was installed, so every installer must run first"},
		{LockDependencies, InstallPreCommit, "the pip path pins each hook's tools to the versions the freeze recorded"},
		{InitGit, InstallHooks, "pre-commit installs its hook into the git repository"},
		{InstallPreCommit, InstallHooks, "the hook runs the configuration pre-commit was given"},
		{WriteGitignore, InitGit, "the repository starts with .env and .venv already ignored"},
		{LockDependencies, Migrate, "a project that cannot be reinstalled should fail before it grows a database"},
		{SetupTailwind, WriteManifest, "the manifest reads the node dependencies out of package.json"},
	}
	for _, c := range constraints {
		if !dependsOn(c.after, c.before) {
			t.Errorf("%q must wait for %q: %s", c.after, c.before, c.why)
		}
	}
	all := Effects()
	for _, e := range all[1:] {
		if !dependsOn(e.ID, CreateVenv) {
			t.Errorf("%q can start before the virtual environment settles the toolchain it renders for", e.ID)
		}
	}
	for _, e := range Phase(all, false) {
		if e.ID != Verify && !dependsOn(Verify, e.ID) {
			t.Errorf("the checks run before %q has finished", e.ID)
		}
	}
}

func TestOnlyTheHooksWaitForThePublishedFolder(t *testing.T) {
	t.Parallel()

	published := map[string]bool{}
	for _, e := range Effects() {
		published[e.ID] = e.Published
	}
	for _, e := range Effects() {
		if e.Published != (e.ID == InstallHooks) {
			t.Errorf("%q published=%v; only the git hooks record the folder's absolute path", e.ID, e.Published)
		}
		for _, need := range e.Needs {
			if published[need] && !e.Published {
				t.Errorf("%q runs before publishing but needs %q, which runs after", e.ID, need)
			}
		}
	}
}

func TestGitTheDatabaseAndTailwindRunBesideTheFiles(t *testing.T) {
	t.Parallel()

	files := []string{WriteProject, WriteSettings, WriteURLs, WriteGitignore, WriteTemplates, WriteAllAuth, WriteSharedApps, WriteApp, WriteDRF, WriteNinja, LockDependencies}
	for _, side := range []string{InitGit, StartDatabase, SetupTailwind} {
		for _, id := range files {
			if dependsOn(id, side) {
				t.Errorf("%q waits for %q, so a slow %s holds up the project files", id, side, side)
			}
		}
	}
}

func TestEachFileGroupIsWrittenByOneDeclaredEffect(t *testing.T) {
	t.Parallel()

	if missing := unwritten(); len(missing) > 0 {
		t.Errorf("no effect declares the %v files", missing)
	}
	for name, spec := range map[string]config.Spec{
		"minimal":    specFor(t, config.Choices{}),
		"everything": everything(t),
		"ninja":      specFor(t, config.Choices{App: "catalog", Stack: config.StackHTMX, AddOns: config.NewSet(config.Ninja)}),
	} {
		if err := Check(spec); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
}

func TestOnlyTheDatabaseDeclaresACompensation(t *testing.T) {
	t.Parallel()

	for _, e := range Effects() {
		if e.Compensate != (e.ID == StartDatabase) {
			t.Errorf("%q compensate=%v; only the database leaves state outside the project folder", e.ID, e.Compensate)
		}
	}
}

func TestThePlanSelectsEffectsByConfiguration(t *testing.T) {
	t.Parallel()

	ids := func(spec config.Spec) []string {
		var out []string
		for _, e := range For(spec) {
			out = append(out, e.ID)
		}
		return out
	}
	minimal := ids(specFor(t, config.Choices{}))
	for _, want := range []string{CreateVenv, InstallPackages, WriteEnv, Migrate, WriteGitignore} {
		if !slices.Contains(minimal, want) {
			t.Errorf("a minimal project skips %q", want)
		}
	}
	for _, unwanted := range []string{StartDatabase, WriteCompose, SetupTailwind, WriteDRF, WriteApp, InitGit} {
		if slices.Contains(minimal, unwanted) {
			t.Errorf("a minimal project runs %q", unwanted)
		}
	}
	full := ids(everything(t))
	for _, want := range []string{StartDatabase, SetupTailwind, WriteAllAuth, InstallPreCommit, WriteApp, WriteDRF, InitGit} {
		if !slices.Contains(full, want) {
			t.Errorf("the everything project skips %q", want)
		}
	}
	if Count(everything(t)) != len(full) {
		t.Errorf("Count disagrees with For")
	}
}

func TestEveryProjectGetsACompleteGitignore(t *testing.T) {
	t.Parallel()

	content := render.RenderStatic("gitignore.tmpl")
	for _, entry := range []string{".env\n", ".venv/", "*.sqlite3", "node_modules/"} {
		if !strings.Contains(content, entry) {
			t.Errorf(".gitignore does not cover %q", entry)
		}
	}
}
