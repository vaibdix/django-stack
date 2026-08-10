package scaffold

import (
	"fmt"

	"github.com/vaibdix/django-cli/internal/config"
	"github.com/vaibdix/django-cli/internal/scaffold/dbdocker"
)

// The scaffolding plan.
//
// Generation used to be one 250-line function with a cyclomatic complexity of
// 60: twenty-odd `if e.Cfg.SetupX { ... }` blocks in a fixed sequence, where the
// sequence itself was the important part and was recorded only in comments
// scattered through the body.
//
// Here the sequence is data. `plan()` is a pure function of the configuration —
// it runs nothing — so the order can be asserted in a test without creating a
// virtualenv or reaching PyPI. That is what TestPlanMatchesCanonicalOrder does.

// step is one unit of scaffolding work.
type step struct {
	// name identifies the step in errors and in the ordering test. These names
	// are a contract: TestPlanMatchesCanonicalOrder pins them.
	name string

	// label is what the user sees on the progress line while the step runs.
	label string

	// when reports whether the step applies to this configuration.
	// A nil when means the step always runs.
	when func(config.Project) bool

	// run performs the work.
	run func(e *Env, projectPath string) error
}

// applies reports whether the step should run for cfg.
func (s step) applies(cfg config.Project) bool {
	return s.when == nil || s.when(cfg)
}

// StepCount returns how many steps will run for cfg, which is exactly how many
// progress updates the generation will emit.
//
// The TUI uses this as the denominator of its progress bar. Previously the TUI
// summed a hand-maintained list of per-feature constants while the engine fired
// progress updates from wherever it happened to feel appropriate — the two
// disagreed in every configuration, and enabling Tailwind fired two updates
// against a budget of one. Deriving both from the same list makes them agree by
// construction rather than by maintenance.
func StepCount(cfg config.Project) int {
	n := 0
	for _, s := range plan() {
		if s.applies(cfg) {
			n++
		}
	}
	return n
}

// Predicates, named so the plan below reads as prose.
func usesDocker(c config.Project) bool      { return c.DatabaseType.UsesDocker() }
func usesPostgresURL(c config.Project) bool { return c.DatabaseType == config.PostgresString }
func wantsGit(c config.Project) bool        { return c.InitializeGit }
func wantsTemplates(c config.Project) bool  { return c.CreateTemplates }
func wantsTailwind(c config.Project) bool   { return c.SetupTailwind }
func wantsHTMX(c config.Project) bool       { return c.SetupHTMX }
func wantsAllAuth(c config.Project) bool    { return c.SetupAllAuth }
func wantsRuff(c config.Project) bool       { return c.SetupRuff }
func wantsMypy(c config.Project) bool       { return c.SetupMypy }
func wantsPreCommit(c config.Project) bool  { return c.SetupPreCommit }
func wantsMultiEnv(c config.Project) bool   { return c.SetupMultiEnv }
func wantsREST(c config.Project) bool       { return c.SetupRestFramework }
func wantsApp(c config.Project) bool        { return c.AppName != "" }
func wantsServer(c config.Project) bool     { return c.RunServer }

// plan returns the ordered steps for a project configuration.
//
// Three orderings are load-bearing. Each is marked at the step that depends on
// the one before it; breaking any of them produces a project that looks
// scaffolded and does not work.
func plan() []step {
	return []step{
		{name: "create virtual environment", label: "Creating virtual environment...", run: (*Env).createVirtualEnvironment},
		{name: "install django", label: "Installing Django...", run: (*Env).installDjango},
		{name: "create django project", label: "Creating Django project...", run: (*Env).createDjangoProject},

		// LOAD-BEARING: the container must exist before .env is written, because
		// .env reuses the password generated for it. Reversed, the generated
		// project holds credentials the running database does not accept.
		{name: "setup docker database", label: "Starting the database container...", when: usesDocker, run: (*Env).startDatabaseContainer},

		{name: "create .env", label: "Creating environment configuration...", run: (*Env).writeEnvFile},
		{name: "create .env.example", label: "Writing .env.example...", run: (*Env).writeEnvExampleFile},
		{name: "write docker-compose.yml", label: "Creating docker-compose.yml...", when: usesDocker, run: (*Env).writeComposeFile},
		{name: "write postgres readme", label: "Writing PostgreSQL setup instructions...", when: usesPostgresURL, run: (*Env).writePostgresReadme},

		{name: "configure django settings", label: "Configuring Django settings...", run: (*Env).configureProjectSettings},
		{name: "setup project urls", label: "Configuring project URLs...", run: (*Env).writeProjectURLs},
		{name: "initialize git", label: "Initializing the git repository...", when: wantsGit, run: (*Env).initGitRepository},
		{name: "setup global templates", label: "Setting up global templates and static files...", when: wantsTemplates, run: (*Env).applyGlobalTemplatesAndSettings},

		{name: "setup tailwind", label: "Setting up Tailwind CSS v4...", when: wantsTailwind, run: (*Env).applyTailwindStep},
		{name: "setup htmx", label: "Setting up HTMX...", when: wantsHTMX, run: (*Env).applyHTMX},
		{name: "setup allauth", label: "Setting up Django AllAuth...", when: wantsAllAuth, run: (*Env).applyAllAuth},
		{name: "setup ruff", label: "Setting up Ruff linter...", when: wantsRuff, run: (*Env).applyRuff},
		{name: "setup mypy", label: "Setting up Mypy type checking...", when: wantsMypy, run: (*Env).applyMypy},

		// LOAD-BEARING: pre-commit's config references the ruff configuration
		// written above, so ruff must have run first.
		{name: "setup pre-commit", label: "Setting up pre-commit hooks...", when: wantsPreCommit, run: (*Env).applyPreCommit},

		{name: "create django app", label: "Creating the Django app...", when: wantsApp, run: (*Env).createDjangoAppStep},
		{name: "setup rest framework", label: "Setting up Django REST Framework...", when: wantsREST, run: (*Env).applyRESTFramework},

		// LOAD-BEARING: multi-env replaces settings.py with a settings/ package,
		// so it must run after every other step that writes to settings.py.
		{name: "setup multi-env", label: "Splitting settings into multi-env layout...", when: wantsMultiEnv, run: (*Env).applyMultiEnvSettings},

		{name: "run migrations", label: "Running database migrations...", run: (*Env).runMigrations},
		{name: "write superuser guide", label: "Writing the superuser guide...", run: (*Env).writeSuperuserGuide},
		{name: "run scaffold formatting", label: "Formatting generated code and templates...", run: (*Env).formatGeneratedCode},
		{name: "report server instructions", label: "Preparing server instructions...", when: wantsServer, run: (*Env).reportServerInstructionsStep},
		{name: "write vscode config", label: "Writing VS Code configuration...", run: (*Env).writeVSCodeConfigStep},
		{name: "write dev start script", label: "Writing the development start script...", run: (*Env).writeDevStartScript},
	}
}

// Adapters, so every step has the same signature. Each wraps a call that either
// needs more than the project path or returns nothing.

// startDatabaseContainer starts the configured database container and records
// its credentials on the config, where the .env writer reads them.
func (e *Env) startDatabaseContainer(projectPath string) error {
	ctx, cancel := e.timeoutContext()
	defer cancel()

	setup, label := dbdocker.SetupPostgres, "PostgreSQL"
	if e.Cfg.DatabaseType == config.MySQLDocker {
		setup, label = dbdocker.SetupMySQL, "MySQL"
	}

	e.Step("Setting up " + label + " with Docker...")
	creds, err := setup(ctx, e.runner(), e, e.Cfg.ProjectName)
	if err != nil {
		return fmt.Errorf("failed to set up %s Docker: %w", label, err)
	}

	switch e.Cfg.DatabaseType {
	case config.PostgresDocker:
		e.Cfg.PostgresDockerPort, e.Cfg.PostgresDockerPassword = creds.Port, creds.Password
	case config.MySQLDocker:
		e.Cfg.MySQLDockerPort, e.Cfg.MySQLDockerPassword = creds.Port, creds.Password
	}

	e.Step(label + " Docker container started!")
	e.Step("Connection: " + creds.ConnectionURL)

	if e.Cfg.DatabaseType == config.MySQLDocker {
		return e.installPackages(projectPath, "mysqlclient", PkgMySQLClient)
	}
	return nil
}

func (e *Env) writeComposeFile(projectPath string) error {
	e.Step("Creating docker-compose.yml...")
	if err := dbdocker.WriteComposeFile(projectPath, e.Cfg.DatabaseType); err != nil {
		return fmt.Errorf("failed to create docker-compose.yml: %w", err)
	}
	e.Step("Created: docker-compose.yml")
	return nil
}

func (e *Env) configureProjectSettings(projectPath string) error {
	return e.configureSettings(e.settingsPath(projectPath))
}

// applyGlobalTemplatesAndSettings writes the templates and then points
// settings.py at them; the two are one step because neither is useful alone.
func (e *Env) applyGlobalTemplatesAndSettings(projectPath string) error {
	if err := e.applyGlobalTemplates(projectPath); err != nil {
		return err
	}

	se, err := NewSettingsEditor(e.settingsPath(projectPath))
	if err != nil {
		return err
	}
	if err := se.ConfigureTemplatesDirs(); err != nil {
		return err
	}
	if err := se.Save(); err != nil {
		return err
	}

	e.Step("Configured settings for global templates and static files.")
	return nil
}

// applyTailwindStep adapts applyTailwind, which reports failures as warnings
// and never fails the scaffold.
func (e *Env) applyTailwindStep(projectPath string) error {
	e.applyTailwind(projectPath)
	return nil
}

func (e *Env) createDjangoAppStep(projectPath string) error {
	if err := e.createDjangoApp(projectPath, e.settingsPath(projectPath)); err != nil {
		return err
	}
	e.Step(fmt.Sprintf("Created Django app '%s' with templates and URLs.", e.Cfg.AppName))
	return nil
}

func (e *Env) reportServerInstructionsStep(string) error {
	e.reportServerInstructions()
	return nil
}

func (e *Env) writeVSCodeConfigStep(projectPath string) error {
	writeVSCodeConfig(projectPath, e.Cfg.SetupTailwind)
	e.Step("VS Code configuration created (.vscode/)")
	return nil
}

// settingsPath is where Django's settings.py lives for this project.
func (e *Env) settingsPath(projectPath string) string {
	return joinPath(projectPath, e.Cfg.ModuleName(), SettingsFile)
}
