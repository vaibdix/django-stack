package plan

import (
	"fmt"
	"time"

	"github.com/vaibdix/django-stack/internal/config"
	"github.com/vaibdix/django-stack/internal/render"
)

const (
	CreateVenv       = "create virtual environment"
	InstallPackages  = "install django"
	SetupTailwind    = "setup tailwind"
	WriteEnv         = "create .env"
	WriteEnvExample  = "create .env.example"
	WriteCompose     = "write docker-compose.yml"
	StartDatabase    = "setup docker database"
	WriteProject     = "create django project"
	WriteSettings    = "configure django settings"
	WriteURLs        = "setup project urls"
	WriteGitignore   = "write .gitignore"
	InitGit          = "initialize git"
	WriteTemplates   = "setup global templates"
	WriteAllAuth     = "setup allauth"
	WriteSharedApps  = "create shared apps"
	WriteApp         = "create django app"
	WriteDRF         = "setup rest framework"
	WriteNinja       = "setup django ninja"
	LockDependencies = "write dependency manifest"
	BuildCSS         = "build tailwind css"
	InstallPreCommit = "setup pre-commit"
	Migrate          = "run migrations"
	WriteReadme      = "write readme"
	WriteWorkspace   = "write vscode config"
	WriteDevScript   = "write dev start script"
	WriteManifest    = "write stack manifest"
	Verify           = "verify project"
	InstallHooks     = "install git hooks"
)

const Concurrency = 4

const (
	commandTimeout = 10 * time.Minute
	writeTimeout   = time.Minute
)

type Effect struct {
	ID         string
	Label      string
	Needs      []string
	Produces   []string
	Timeout    time.Duration
	Compensate bool
	Published  bool
	When       func(config.Spec) bool
}

func (e Effect) Applies(s config.Spec) bool { return e.When == nil || e.When(s) }

func Group(g render.Group) string { return "files:" + string(g) }

func usesDocker(s config.Spec) bool { return s.Database().UsesDocker() }

func wantsApp(s config.Spec) bool { return s.App().Ok() }

func has(addOn config.AddOn) func(config.Spec) bool {
	return func(s config.Spec) bool { return s.Has(addOn) }
}

func Effects() []Effect {
	return []Effect{
		{ID: CreateVenv, Label: "Creating virtual environment...", Produces: []string{"venv"}, Timeout: commandTimeout},
		{ID: InstallPackages, Label: "Installing Django...", Needs: []string{CreateVenv},
			Produces: []string{"packages"}, Timeout: commandTimeout},
		{ID: SetupTailwind, Label: "Setting up Tailwind CSS " + render.TailwindVersion() + "...", When: config.Spec.UsesTailwind,
			Needs: []string{CreateVenv}, Produces: []string{Group(render.GroupTailwind), "node packages"}, Timeout: commandTimeout},
		{ID: WriteEnv, Label: "Creating environment configuration...", Needs: []string{CreateVenv},
			Produces: []string{Group(render.GroupEnv)}, Timeout: writeTimeout},
		{ID: WriteEnvExample, Label: "Writing .env.example...", Needs: []string{CreateVenv},
			Produces: []string{Group(render.GroupEnvExample)}, Timeout: writeTimeout},
		{ID: WriteCompose, Label: "Creating docker-compose.yml...", When: usesDocker, Needs: []string{CreateVenv},
			Produces: []string{Group(render.GroupCompose)}, Timeout: writeTimeout},
		{ID: StartDatabase, Label: "Starting the database with docker compose...", When: usesDocker,
			Needs: []string{WriteEnv, WriteCompose}, Produces: []string{"database"}, Timeout: commandTimeout, Compensate: true},
		{ID: WriteProject, Label: "Creating Django project...", Needs: []string{InstallPackages},
			Produces: []string{Group(render.GroupProject)}, Timeout: writeTimeout},
		{ID: WriteSettings, Label: "Configuring Django settings...", Needs: []string{WriteProject},
			Produces: []string{Group(render.GroupSettings)}, Timeout: writeTimeout},
		{ID: WriteURLs, Label: "Configuring project URLs...", Needs: []string{WriteSettings},
			Produces: []string{Group(render.GroupURLs)}, Timeout: writeTimeout},
		{ID: WriteGitignore, Label: "Writing .gitignore...", Needs: []string{WriteURLs},
			Produces: []string{Group(render.GroupGitignore)}, Timeout: writeTimeout},
		{ID: InitGit, Label: "Initializing the git repository...", When: has(config.Git), Needs: []string{WriteGitignore},
			Produces: []string{"git repository"}, Timeout: commandTimeout},
		{ID: WriteTemplates, Label: "Setting up global templates and static files...", When: config.Spec.CreateTemplates,
			Needs: []string{WriteGitignore}, Produces: []string{Group(render.GroupTemplates)}, Timeout: writeTimeout},
		{ID: WriteAllAuth, Label: "Setting up Django AllAuth...", When: has(config.AllAuth), Needs: []string{WriteTemplates},
			Produces: []string{Group(render.GroupAllAuth)}, Timeout: writeTimeout},
		{ID: WriteSharedApps, Label: "Creating the core and accounts apps...", Needs: []string{WriteAllAuth},
			Produces: []string{Group(render.GroupSharedApps)}, Timeout: writeTimeout},
		{ID: WriteApp, Label: "Creating the Django app...", When: wantsApp, Needs: []string{WriteSharedApps},
			Produces: []string{Group(render.GroupApp)}, Timeout: writeTimeout},
		{ID: WriteDRF, Label: "Setting up Django REST Framework...", When: has(config.RestFramework), Needs: []string{WriteApp},
			Produces: []string{Group(render.GroupAPI)}, Timeout: writeTimeout},
		{ID: WriteNinja, Label: "Setting up Django Ninja...", When: has(config.Ninja), Needs: []string{WriteDRF},
			Produces: []string{Group(render.GroupAPI)}, Timeout: writeTimeout},
		{ID: LockDependencies, Label: "Recording dependencies and locking...", Needs: []string{WriteNinja},
			Produces: []string{"pyproject.toml", "lock", "requirements"}, Timeout: commandTimeout},
		{ID: BuildCSS, Label: "Compiling Tailwind CSS...", When: config.Spec.UsesTailwind,
			Needs: []string{SetupTailwind, LockDependencies}, Produces: []string{"stylesheet"}, Timeout: commandTimeout},
		{ID: InstallPreCommit, Label: "Setting up pre-commit hooks...", When: has(config.PreCommit),
			Needs: []string{LockDependencies, InitGit}, Produces: []string{".pre-commit-config.yaml"}, Timeout: writeTimeout},
		{ID: Migrate, Label: "Running database migrations...", Needs: []string{InstallPreCommit, WriteEnv, StartDatabase},
			Produces: []string{"database schema"}, Timeout: commandTimeout},
		{ID: WriteReadme, Label: "Writing README.md...", Needs: []string{Migrate},
			Produces: []string{Group(render.GroupReadme)}, Timeout: writeTimeout},
		{ID: WriteWorkspace, Label: "Writing VS Code configuration...", Needs: []string{WriteReadme},
			Produces: []string{Group(render.GroupWorkspace)}, Timeout: writeTimeout},
		{ID: WriteDevScript, Label: "Writing the development start script...", Needs: []string{WriteWorkspace},
			Produces: []string{Group(render.GroupScripts)}, Timeout: writeTimeout},
		{ID: WriteManifest, Label: "Recording the project setup...", Needs: []string{WriteDevScript, SetupTailwind, BuildCSS, WriteEnvExample},
			Produces: []string{"django-stack.json"}, Timeout: writeTimeout},
		{ID: Verify, Label: "Checking the project passes its own checks...", Needs: []string{WriteManifest},
			Timeout: commandTimeout},
		{ID: InstallHooks, Label: "Installing the git hooks...", When: has(config.PreCommit), Needs: []string{InstallPreCommit, InitGit},
			Produces: []string{"git hooks"}, Timeout: commandTimeout, Published: true},
	}
}

func Phase(effects []Effect, published bool) []Effect {
	var phase []Effect
	for _, e := range effects {
		if e.Published == published {
			phase = append(phase, e)
		}
	}
	return phase
}

func For(s config.Spec) []Effect {
	var applied []Effect
	for _, e := range Effects() {
		if e.Applies(s) {
			applied = append(applied, e)
		}
	}
	return applied
}

func Count(s config.Spec) int { return len(For(s)) }

func Check(s config.Spec) error {
	if missing := unwritten(); len(missing) > 0 {
		return fmt.Errorf("plan: no effect declares the %v files", missing)
	}
	produced := map[string]string{}
	for _, e := range For(s) {
		for _, p := range e.Produces {
			if other, ok := produced[p]; ok {
				return fmt.Errorf("plan: %s and %s both produce %s", other, e.ID, p)
			}
			produced[p] = e.ID
		}
	}
	return nil
}

func unwritten() []render.Group {
	declared := map[string]bool{}
	for _, e := range Effects() {
		for _, p := range e.Produces {
			declared[p] = true
		}
	}
	var missing []render.Group
	for _, g := range render.Groups() {
		if !declared[Group(g)] {
			missing = append(missing, g)
		}
	}
	return missing
}
