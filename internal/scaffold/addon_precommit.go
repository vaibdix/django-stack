package scaffold

import (
	"context"
	"fmt"
	"slices"

	"github.com/vaibdix/django-stack/internal/config"
	"github.com/vaibdix/django-stack/internal/pyenv"
	"github.com/vaibdix/django-stack/internal/render"
)

func (e *Env) applyPreCommit(ctx context.Context) error {
	config, err := e.generatePreCommitConfig(ctx, e.dir)
	if err != nil {
		return err
	}

	if err := e.write(render.PreCommitConfigFile, []byte(config), render.FileMode); err != nil {
		return err
	}
	e.Step("Created: " + render.PreCommitConfigFile)

	pythonPath := pyenv.InterpreterPath(e.dir)
	ctx, cancel := context.WithTimeout(ctx, defaultCommandTimeout)
	defer cancel()

	if _, err := e.runIn(ctx, e.dir, pythonPath, "-m", "pre_commit", "install"); err != nil {
		return fmt.Errorf("install the pre-commit hooks: %w", err)
	}

	e.Step("Pre-commit hooks installed")
	return nil
}

func (e *Env) generatePreCommitConfig(ctx context.Context, projectPath string) (string, error) {
	data := render.PreCommitData{
		RuffEnabled: e.Cfg.Has(config.Ruff),
		MypyEnabled: e.Cfg.Has(config.Mypy),
		UVAvailable: e.Toolchain == pyenv.UV,
		Templates:   e.Cfg.CreateTemplates(),
		DjangoTrack: e.Cfg.Django().Series(),
	}
	versions, err := e.freezeVersions(ctx, projectPath)
	if err != nil {
		return "", err
	}
	hookTools := []string{render.PkgRuff, render.PkgDjLint, render.PkgPreCommit}
	var mypyDeps []string
	for _, spec := range append(append([]string{}, e.packages.Runtime...), e.packages.Dev...) {
		if !slices.Contains(hookTools, render.DistributionName(spec)) {
			mypyDeps = append(mypyDeps, spec)
		}
	}
	if data.RuffDeps, err = render.ExactPins([]string{render.PkgRuff}, versions, data.RuffEnabled); err != nil {
		return "", err
	}
	if data.DjLintDeps, err = render.ExactPins([]string{render.PkgDjLint}, versions, data.Templates); err != nil {
		return "", err
	}
	if data.MypyDeps, err = render.ExactPins(mypyDeps, versions, data.MypyEnabled); err != nil {
		return "", err
	}
	return render.RenderPreCommitConfig(data), nil
}
