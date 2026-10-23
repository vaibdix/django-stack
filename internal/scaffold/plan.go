package scaffold

import (
	"context"
	"fmt"

	"github.com/vaibdix/django-stack/internal/config"
	"github.com/vaibdix/django-stack/internal/dbdocker"
)

type step struct {
	name string

	label string

	when func(config.Choices) bool

	run func(e *Env, ctx context.Context) error
}

func (s step) applies(cfg config.Choices) bool {
	return s.when == nil || s.when(cfg)
}

func StepCount(cfg config.Choices) int {
	n := 0
	for _, s := range plan() {
		if s.applies(cfg) {
			n++
		}
	}
	return n
}

func usesDocker(c config.Choices) bool { return c.Database.UsesDocker() }

func has(addOn config.AddOn) func(config.Choices) bool {
	return func(c config.Choices) bool { return c.Has(addOn) }
}

func wantsApp(c config.Choices) bool { return c.App != "" }

func plan() []step {
	return []step{
		{name: "create virtual environment", label: "Creating virtual environment...", run: (*Env).createVirtualEnvironment},
		{name: "install django", label: "Installing Django...", run: (*Env).installDjango},
		{name: "create django project", label: "Creating Django project...", run: (*Env).createDjangoProject},

		{name: "choose database credentials", label: "Choosing database credentials and a free port...", when: usesDocker, run: (*Env).prepareDatabaseCredentials},

		{name: "create .env", label: "Creating environment configuration...", run: (*Env).writeEnvFile},
		{name: "create .env.example", label: "Writing .env.example...", run: (*Env).writeEnvExampleFile},
		{name: "write docker-compose.yml", label: "Creating docker-compose.yml...", when: usesDocker, run: (*Env).writeComposeFile},
		{name: "setup docker database", label: "Starting the database with docker compose...", when: usesDocker, run: (*Env).startDatabaseContainer},

		{name: "configure django settings", label: "Configuring Django settings...", run: (*Env).configureProjectSettings},
		{name: "setup project urls", label: "Configuring project URLs...", run: (*Env).writeProjectURLs},
		{name: "write .gitignore", label: "Writing .gitignore...", run: (*Env).writeGitignore},
		{name: "initialize git", label: "Initializing the git repository...", when: has(config.Git), run: (*Env).initGitRepository},
		{name: "setup global templates", label: "Setting up global templates and static files...", when: config.Choices.CreateTemplates, run: (*Env).applyGlobalTemplatesAndSettings},

		{name: "setup tailwind", label: "Setting up Tailwind CSS " + tailwindVersion + "...", when: config.Choices.UsesTailwind, run: (*Env).applyTailwind},
		{name: "setup htmx", label: "Setting up HTMX...", when: config.Choices.UsesHTMX, run: (*Env).applyHTMX},
		{name: "setup allauth", label: "Setting up Django AllAuth...", when: has(config.AllAuth), run: (*Env).applyAllAuth},
		{name: "setup structlog", label: "Setting up structured logging...", when: has(config.Structlog), run: (*Env).applyStructlog},
		{name: "setup sentry", label: "Setting up Sentry error tracking...", when: has(config.Sentry), run: (*Env).applySentry},
		{name: "setup ruff", label: "Setting up Ruff linter...", when: has(config.Ruff), run: (*Env).applyRuff},
		{name: "setup mypy", label: "Setting up Mypy type checking...", when: has(config.Mypy), run: (*Env).applyMypy},

		{name: "create shared apps", label: "Creating the core and accounts apps...", run: (*Env).createAccountsApp},
		{name: "create django app", label: "Creating the Django app...", when: wantsApp, run: (*Env).createDjangoAppStep},
		{name: "setup rest framework", label: "Setting up Django REST Framework...", when: has(config.RestFramework), run: (*Env).applyRESTFramework},
		{name: "setup django ninja", label: "Setting up Django Ninja...", when: has(config.Ninja), run: (*Env).applyNinja},

		{name: "build tailwind css", label: "Compiling Tailwind CSS...", when: config.Choices.UsesTailwind, run: (*Env).buildTailwindCSS},

		{name: "setup multi-env", label: "Splitting settings into multi-env layout...", when: has(config.MultiEnv), run: (*Env).applyMultiEnvSettings},

		{name: "write dependency manifest", label: "Recording dependencies and locking...", run: (*Env).writeDependencyManifest},
		{name: "setup pre-commit", label: "Setting up pre-commit hooks...", when: has(config.PreCommit), run: (*Env).applyPreCommit},

		{name: "run migrations", label: "Running database migrations...", run: (*Env).runMigrations},
		{name: "write readme", label: "Writing README.md...", run: (*Env).writeReadme},
		{name: "run scaffold formatting", label: "Formatting generated code and templates...", run: (*Env).formatGeneratedCode},
		{name: "write vscode config", label: "Writing VS Code configuration...", run: (*Env).writeVSCodeConfigStep},
		{name: "write dev start script", label: "Writing the development start script...", run: (*Env).writeDevStartScript},

		{name: "write stack manifest", label: "Recording the project setup...", run: (*Env).writeStackManifest},

		{name: "verify project", label: "Checking the project passes its own checks...", run: (*Env).verifyProject},
	}
}

func (e *Env) prepareDatabaseCredentials(ctx context.Context) error {
	creds, err := dbdocker.NewCredentials(e.cfg.Database, e.cfg.Name)
	if err != nil {
		return err
	}

	e.resolved.Database = config.DatabaseCredentials{Port: creds.Port, Password: creds.Password, RootPassword: creds.RootPassword}
	e.Step("Database " + creds.DatabaseName + " will listen on 127.0.0.1:" + creds.Port)
	return nil
}

func (e *Env) startDatabaseContainer(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, defaultCommandTimeout)
	defer cancel()

	if err := dbdocker.Start(ctx, e.commandRunner(), e, e.dir, e.cfg.Name, e.cfg.Database); err != nil {
		return fmt.Errorf("failed to start the database: %w", err)
	}
	e.onRollback(func(ctx context.Context) error {
		return dbdocker.Stop(ctx, e.commandRunner(), e.dir)
	})

	return nil
}

func (e *Env) writeComposeFile(ctx context.Context) error {
	if err := dbdocker.WriteComposeFile(e.dir, e.cfg.Name, e.cfg.Database); err != nil {
		return fmt.Errorf("failed to create %s: %w", dbdocker.ComposeFile, err)
	}
	e.Step("Created: " + dbdocker.ComposeFile)
	return nil
}

func (e *Env) configureProjectSettings(ctx context.Context) error {
	if err := e.writeConfModule(e.dir); err != nil {
		return err
	}
	return e.configureSettings(e.settingsPath(e.dir))
}

func (e *Env) applyGlobalTemplatesAndSettings(ctx context.Context) error {
	if err := e.applyGlobalTemplates(e.dir); err != nil {
		return err
	}

	se, err := newSettingsEditor(e.settingsPath(e.dir))
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

func (e *Env) createDjangoAppStep(ctx context.Context) error {
	if err := e.createDjangoApp(ctx, e.dir, e.settingsPath(e.dir)); err != nil {
		return err
	}
	e.Step(fmt.Sprintf("Created Django app '%s' with templates and URLs.", e.cfg.App))
	return nil
}

func (e *Env) writeVSCodeConfigStep(ctx context.Context) error {
	if err := writeVSCodeConfig(e.dir, e.cfg, e.node); err != nil {
		return err
	}
	e.Step("VS Code configuration created (.vscode/)")
	return nil
}

func (e *Env) settingsPath(projectPath string) string {
	return joinPath(projectPath, e.cfg.ModuleName(), settingsFile)
}
