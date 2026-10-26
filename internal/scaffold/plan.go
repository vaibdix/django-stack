package scaffold

import (
	"context"
	"fmt"

	"github.com/vaibdix/django-stack/internal/config"
	"github.com/vaibdix/django-stack/internal/dbdocker"
	"github.com/vaibdix/django-stack/internal/graph"
)

type step struct {
	name string

	label string

	when func(config.Spec) bool

	run func(e *Env, ctx context.Context) error

	side bool

	needs []string
}

func (s step) applies(cfg config.Spec) bool {
	return s.when == nil || s.when(cfg)
}

func StepCount(cfg config.Spec) int {
	count := 0
	for _, s := range plan() {
		if s.applies(cfg) {
			count++
		}
	}
	return count
}

func nodes(cfg config.Spec) []graph.Node[*Env] {
	var all []graph.Node[*Env]
	previous := ""
	for _, s := range plan() {
		needs := s.needs
		if !s.side && previous != "" {
			needs = append([]string{previous}, needs...)
		}
		if !s.side {
			previous = s.name
		}
		all = append(all, graph.Node[*Env]{Name: s.name, Needs: needs, Skip: !s.applies(cfg), Run: s.run})
	}
	return all
}

func labels() map[string]string {
	all := map[string]string{}
	for _, s := range plan() {
		all[s.name] = s.label
	}
	return all
}

func usesDocker(c config.Spec) bool { return c.Database().UsesDocker() }

func has(addOn config.AddOn) func(config.Spec) bool {
	return func(c config.Spec) bool { return c.Has(addOn) }
}

func wantsApp(c config.Spec) bool { return c.App().Ok() }

func plan() []step {
	return []step{
		{name: "create virtual environment", label: "Creating virtual environment...", run: (*Env).createVirtualEnvironment},
		{name: "install django", label: "Installing Django...", run: (*Env).installDjango},
		{name: "setup tailwind", label: "Setting up Tailwind CSS " + tailwindVersion() + "...", when: config.Spec.UsesTailwind, run: (*Env).applyTailwind, side: true,
			needs: []string{"create virtual environment"}},
		{name: "create .env", label: "Creating environment configuration...", run: (*Env).writeEnvFile, side: true,
			needs: []string{"create virtual environment"}},
		{name: "create .env.example", label: "Writing .env.example...", run: (*Env).writeEnvExampleFile, side: true,
			needs: []string{"create virtual environment"}},
		{name: "write docker-compose.yml", label: "Creating docker-compose.yml...", when: usesDocker, run: (*Env).writeComposeFile, side: true,
			needs: []string{"create virtual environment"}},
		{name: "setup docker database", label: "Starting the database with docker compose...", when: usesDocker, run: (*Env).startDatabaseContainer, side: true,
			needs: []string{"create .env", "write docker-compose.yml"}},
		{name: "create django project", label: "Creating Django project...", run: (*Env).createDjangoProject},
		{name: "configure django settings", label: "Configuring Django settings...", run: (*Env).configureProjectSettings},
		{name: "setup project urls", label: "Configuring project URLs...", run: (*Env).writeProjectURLs},
		{name: "write .gitignore", label: "Writing .gitignore...", run: (*Env).writeGitignore},
		{name: "initialize git", label: "Initializing the git repository...", when: has(config.Git), run: (*Env).initGitRepository},
		{name: "setup global templates", label: "Setting up global templates and static files...", when: config.Spec.CreateTemplates, run: (*Env).applyGlobalTemplatesAndSettings},
		{name: "setup allauth", label: "Setting up Django AllAuth...", when: has(config.AllAuth), run: (*Env).applyAllAuth},
		{name: "create shared apps", label: "Creating the core and accounts apps...", run: (*Env).createAccountsApp},
		{name: "create django app", label: "Creating the Django app...", when: wantsApp, run: (*Env).createDjangoAppStep},
		{name: "setup rest framework", label: "Setting up Django REST Framework...", when: has(config.RestFramework), run: (*Env).applyRESTFramework},
		{name: "setup django ninja", label: "Setting up Django Ninja...", when: has(config.Ninja), run: (*Env).applyNinja},
		{name: "write dependency manifest", label: "Recording dependencies and locking...", run: (*Env).writeDependencyManifest},
		{name: "build tailwind css", label: "Compiling Tailwind CSS...", when: config.Spec.UsesTailwind, run: (*Env).buildTailwindCSS, side: true,
			needs: []string{"setup tailwind", "write dependency manifest"}},
		{name: "setup pre-commit", label: "Setting up pre-commit hooks...", when: has(config.PreCommit), run: (*Env).applyPreCommit},
		{name: "run migrations", label: "Running database migrations...", run: (*Env).runMigrations,
			needs: []string{"create .env", "setup docker database"}},
		{name: "write readme", label: "Writing README.md...", run: (*Env).writeReadme},
		{name: "write vscode config", label: "Writing VS Code configuration...", run: (*Env).writeVSCodeConfigStep},
		{name: "write dev start script", label: "Writing the development start script...", run: (*Env).writeDevStartScript},
		{name: "write stack manifest", label: "Recording the project setup...", run: (*Env).writeStackManifest,
			needs: []string{"setup tailwind", "build tailwind css", "create .env.example"}},
		{name: "verify project", label: "Checking the project passes its own checks...", run: (*Env).verifyProject},
	}
}

func (e *Env) prepareDatabaseCredentials() error {
	creds, err := dbdocker.NewCredentials(e.cfg.Database(), e.cfg.Name().String())
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

	if err := dbdocker.Start(ctx, e.commandRunner(), e, e.dir, e.cfg.Name().String(), e.cfg.Database()); err != nil {
		return fmt.Errorf("start the database: %w", err)
	}
	e.onRollback(func(ctx context.Context) error {
		return dbdocker.Stop(ctx, e.commandRunner(), e.dir)
	})

	return nil
}

func (p project) composeFiles() (fileSet, error) {
	files := fileSet{}
	if !p.cfg.Database().UsesDocker() {
		return files, nil
	}
	content, err := dbdocker.Compose(p.cfg.Name().String(), p.cfg.Database())
	if err != nil {
		return nil, err
	}
	files.add(dbdocker.ComposeFile, content, fileMode)
	return files, nil
}

func (e *Env) writeComposeFile(ctx context.Context) error {
	files, err := e.composeFiles()
	if err != nil {
		return err
	}
	if err := e.emit(files); err != nil {
		return err
	}
	e.Step("Created: " + dbdocker.ComposeFile)
	return nil
}

func (e *Env) configureProjectSettings(ctx context.Context) error {
	if err := e.writeConfModule(); err != nil {
		return err
	}
	if err := e.emit(e.settingsFiles()); err != nil {
		return err
	}
	if err := e.emit(e.typecheckFiles()); err != nil {
		return err
	}
	e.Step("Django settings configured.")
	if e.cfg.Has(config.Pydantic) {
		e.Step("Environment variables configured via pydantic-settings.")
	} else {
		e.Step("Environment variables configured via django-environ.")
	}
	if e.split() {
		e.Step("Settings split into settings/base.py, local.py, production.py, test.py")
	}
	return nil
}

func (e *Env) applyGlobalTemplatesAndSettings(ctx context.Context) error {
	return e.applyGlobalTemplates()
}

func (e *Env) createDjangoAppStep(ctx context.Context) error {
	if err := e.emit(e.appFiles()); err != nil {
		return err
	}
	e.Step(fmt.Sprintf("Created Django app '%s' with templates and URLs.", e.cfg.App().String()))
	return nil
}

func (e *Env) writeVSCodeConfigStep(ctx context.Context) error {
	if err := e.writeVSCodeConfig(); err != nil {
		return err
	}
	e.Step("VS Code configuration created (.vscode/)")
	return nil
}
