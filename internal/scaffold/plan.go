package scaffold

import (
	"context"
	"fmt"
	"slices"

	"github.com/vaibdix/django-stack/internal/config"
	"github.com/vaibdix/django-stack/internal/dbdocker"
	"github.com/vaibdix/django-stack/internal/graph"
	"github.com/vaibdix/django-stack/internal/plan"
	"github.com/vaibdix/django-stack/internal/render"
)

type effectRun struct {
	run        func(e *Env, ctx context.Context) error
	compensate func(e *Env, ctx context.Context) error
}

func runners() map[string]effectRun {
	return map[string]effectRun{
		plan.CreateVenv:       {run: (*Env).createVirtualEnvironment},
		plan.InstallPackages:  {run: (*Env).installDjango},
		plan.SetupTailwind:    {run: (*Env).applyTailwind},
		plan.WriteEnv:         {run: (*Env).writeEnvFile},
		plan.WriteEnvExample:  {run: (*Env).writeEnvExampleFile},
		plan.WriteCompose:     {run: (*Env).writeComposeFile},
		plan.StartDatabase:    {run: (*Env).startDatabaseContainer, compensate: (*Env).stopDatabaseContainer},
		plan.WriteProject:     {run: (*Env).createDjangoProject},
		plan.WriteSettings:    {run: (*Env).configureProjectSettings},
		plan.WriteURLs:        {run: (*Env).writeProjectURLs},
		plan.WriteGitignore:   {run: (*Env).writeGitignore},
		plan.InitGit:          {run: (*Env).initGitRepository},
		plan.WriteTemplates:   {run: (*Env).applyGlobalTemplatesAndSettings},
		plan.WriteAllAuth:     {run: (*Env).applyAllAuth},
		plan.WriteSharedApps:  {run: (*Env).createAccountsApp},
		plan.WriteApp:         {run: (*Env).createDjangoAppStep},
		plan.WriteDRF:         {run: (*Env).applyRESTFramework},
		plan.WriteNinja:       {run: (*Env).applyNinja},
		plan.LockDependencies: {run: (*Env).writeDependencyManifest},
		plan.BuildCSS:         {run: (*Env).buildTailwindCSS},
		plan.InstallPreCommit: {run: (*Env).applyPreCommit},
		plan.Migrate:          {run: (*Env).runMigrations},
		plan.WriteReadme:      {run: (*Env).writeReadme},
		plan.WriteWorkspace:   {run: (*Env).writeVSCodeConfigStep},
		plan.WriteDevScript:   {run: (*Env).writeDevStartScript},
		plan.WriteManifest:    {run: (*Env).writeStackManifest},
		plan.Verify:           {run: (*Env).verifyProject},
		plan.InstallHooks:     {run: (*Env).installGitHooks},
	}
}

func (e *Env) nodes(published bool) ([]graph.Node[*Env], error) {
	if err := plan.Check(e.Cfg); err != nil {
		return nil, err
	}
	run := runners()
	if all := plan.Effects(); len(all) != len(run) {
		return nil, fmt.Errorf("the plan has %d effects but the executor runs %d", len(all), len(run))
	}
	effects := plan.Phase(plan.Effects(), published)
	inPhase := map[string]bool{}
	for _, effect := range effects {
		inPhase[effect.ID] = true
	}
	nodes := make([]graph.Node[*Env], 0, len(effects))
	for _, effect := range effects {
		r, ok := run[effect.ID]
		if !ok {
			return nil, fmt.Errorf("the executor cannot run %q", effect.ID)
		}
		if effect.Compensate != (r.compensate != nil) {
			return nil, fmt.Errorf("%q declares compensate=%v, but the executor disagrees", effect.ID, effect.Compensate)
		}
		nodes = append(nodes, graph.Node[*Env]{
			Name:    effect.ID,
			Needs:   slices.DeleteFunc(slices.Clone(effect.Needs), func(need string) bool { return !inPhase[need] }),
			Skip:    !effect.Applies(e.Cfg),
			Timeout: effect.Timeout,
			Run:     effectRunner(effect, r),
		})
	}
	return nodes, nil
}

func effectRunner(effect plan.Effect, r effectRun) func(*Env, context.Context) error {
	return func(e *Env, ctx context.Context) error {
		if err := r.run(e, withEffect(ctx, effect)); err != nil {
			return err
		}
		if r.compensate != nil {
			e.onRollback(func(ctx context.Context) error { return r.compensate(e, ctx) })
		}
		return nil
	}
}

type effectKey struct{}

func withEffect(ctx context.Context, effect plan.Effect) context.Context {
	return context.WithValue(ctx, effectKey{}, effect)
}

func runningEffect(ctx context.Context) (plan.Effect, bool) {
	effect, ok := ctx.Value(effectKey{}).(plan.Effect)
	return effect, ok
}

func labels() map[string]string {
	all := map[string]string{}
	for _, effect := range plan.Effects() {
		all[effect.ID] = effect.Label
	}
	return all
}

func (e *Env) prepareDatabaseCredentials() error {
	creds, err := dbdocker.NewCredentials(e.Cfg.Database(), e.Cfg.Name().String(), e.randomness(), e.machine().Docker.FreePort)
	if err != nil {
		return err
	}

	e.Resolved.Database = config.DatabaseCredentials{Port: creds.Port, Password: creds.Password, RootPassword: creds.RootPassword}
	e.Step("Database " + creds.DatabaseName + " will listen on 127.0.0.1:" + creds.Port)
	return nil
}

func (e *Env) startDatabaseContainer(ctx context.Context) error {
	if err := e.machine().Docker.Start(ctx, e, e.dir, e.Cfg.Name().String(), e.Cfg.Database()); err != nil {
		return fmt.Errorf("start the database: %w", err)
	}
	return nil
}

func (e *Env) stopDatabaseContainer(ctx context.Context) error {
	return e.machine().Docker.Stop(ctx, e.dir)
}

func (e *Env) writeComposeFile(ctx context.Context) error {
	if err := e.emitGroup(ctx, render.GroupCompose); err != nil {
		return err
	}
	e.Step("Created: " + dbdocker.ComposeFile)
	return nil
}

func (e *Env) configureProjectSettings(ctx context.Context) error {
	if err := e.emitGroup(ctx, render.GroupSettings); err != nil {
		return err
	}
	if e.Cfg.Has(config.Pydantic) {
		e.Step("Created: " + e.Cfg.ModuleName() + "/" + render.ConfFile + " (typed settings validated at startup)")
	}
	e.Step("Django settings configured.")
	if e.Cfg.Has(config.Pydantic) {
		e.Step("Environment variables configured via pydantic-settings.")
	} else {
		e.Step("Environment variables configured via django-environ.")
	}
	if e.Split() {
		e.Step("Settings split into settings/base.py, local.py, production.py, test.py")
	}
	return nil
}

func (e *Env) applyGlobalTemplatesAndSettings(ctx context.Context) error {
	return e.applyGlobalTemplates(ctx)
}

func (e *Env) createDjangoAppStep(ctx context.Context) error {
	if err := e.emitGroup(ctx, render.GroupApp); err != nil {
		return err
	}
	e.Step(fmt.Sprintf("Created Django app '%s' with templates and URLs.", e.Cfg.App().String()))
	return nil
}

func (e *Env) writeVSCodeConfigStep(ctx context.Context) error {
	if err := e.writeVSCodeConfig(ctx); err != nil {
		return err
	}
	e.Step("VS Code configuration created (.vscode/)")
	return nil
}
