package scaffold

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/vaibdix/django-stack/internal/config"
	"github.com/vaibdix/django-stack/internal/pyenv"
)

func (e *Env) applyPreCommit(ctx context.Context) error {
	config, err := e.generatePreCommitConfig(ctx, e.dir)
	if err != nil {
		return err
	}

	configPath := filepath.Join(e.dir, preCommitConfigFile)
	if err := os.WriteFile(configPath, []byte(config), 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", preCommitConfigFile, err)
	}
	e.Step("Created: " + preCommitConfigFile)

	pythonPath := pyenv.InterpreterPath(e.dir)
	ctx, cancel := context.WithTimeout(ctx, defaultCommandTimeout)
	defer cancel()

	if _, err := e.runIn(ctx, e.dir, pythonPath, "-m", "pre_commit", "install"); err != nil {
		return fmt.Errorf("failed to install the pre-commit hooks: %w", err)
	}

	e.Step("Pre-commit hooks installed")
	return nil
}

const (
	preCommitHooksRev = "v6.0.0"
	djangoUpgradeRev  = "1.33.0"
)

type preCommitData struct {
	RuffEnabled       bool
	MypyEnabled       bool
	UVAvailable       bool
	Templates         bool
	DjangoTrack       string
	PreCommitHooksRev string
	DjangoUpgradeRev  string
	Language          string
	MypyEntry         string
	RuffCheckEntry    string
	RuffFormatEntry   string
	DjLintEntry       string
	RuffDeps          []string
	DjLintDeps        []string
	MypyDeps          []string
}

func (e *Env) generatePreCommitConfig(ctx context.Context, projectPath string) (string, error) {
	data := preCommitData{
		RuffEnabled: e.cfg.Has(config.Ruff),
		MypyEnabled: e.cfg.Has(config.Mypy),
		UVAvailable: e.toolchain == pyenv.UV,
		Templates:   e.cfg.CreateTemplates(),
		DjangoTrack: config.DjangoSeries(e.cfg.Django),
	}
	versions, err := e.freezeVersions(ctx, projectPath)
	if err != nil {
		return "", err
	}
	hookTools := []string{pkgRuff, pkgDjLint, pkgPreCommit}
	var mypyDeps []string
	for _, spec := range append(append([]string{}, e.packages.runtime...), e.packages.dev...) {
		if !slices.Contains(hookTools, distributionName(spec)) {
			mypyDeps = append(mypyDeps, spec)
		}
	}
	if data.RuffDeps, err = exactPins([]string{pkgRuff}, versions, data.RuffEnabled); err != nil {
		return "", err
	}
	if data.DjLintDeps, err = exactPins([]string{pkgDjLint}, versions, data.Templates); err != nil {
		return "", err
	}
	if data.MypyDeps, err = exactPins(mypyDeps, versions, data.MypyEnabled); err != nil {
		return "", err
	}
	return renderPreCommitConfig(data), nil
}

func exactPins(specs []string, versions map[string]string, wanted bool) ([]string, error) {
	if !wanted {
		return nil, nil
	}
	pins := make([]string, 0, len(specs))
	for _, spec := range specs {
		version, ok := installedVersion(spec, versions)
		if !ok {
			return nil, fmt.Errorf("%s was installed but the package freeze does not list it", distributionName(spec))
		}
		requirement := spec
		if i := strings.IndexAny(requirement, "<>=!~;"); i != -1 {
			requirement = requirement[:i]
		}
		pins = append(pins, strings.TrimSpace(requirement)+"=="+version)
	}
	slices.Sort(pins)
	return slices.Compact(pins), nil
}

func renderPreCommitConfig(data preCommitData) string {
	data.PreCommitHooksRev = preCommitHooksRev
	data.DjangoUpgradeRev = djangoUpgradeRev
	data.Language = "python"
	data.MypyEntry = "mypy ."
	data.RuffCheckEntry = "ruff check --fix"
	data.RuffFormatEntry = "ruff format"
	data.DjLintEntry = "djlint --lint --profile=django"
	return render("pre-commit-config.yaml.tmpl", data)
}
