package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/vaibdix/django-stack/internal/config"
	"github.com/vaibdix/django-stack/internal/scaffold/pyenv"
)

func (e *Env) applyPreCommit(projectPath string) error {
	config, err := e.generatePreCommitConfig(projectPath)
	if err != nil {
		return err
	}

	configPath := filepath.Join(projectPath, PreCommitConfigFile)
	if err := os.WriteFile(configPath, []byte(config), 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", PreCommitConfigFile, err)
	}
	e.Step("Created: " + PreCommitConfigFile)

	pythonPath := pyenv.InterpreterPath(projectPath)
	ctx, cancel := e.timeoutContext()
	defer cancel()

	if _, err := e.runIn(ctx, projectPath, pythonPath, "-m", "pre_commit", "install"); err != nil {
		return fmt.Errorf("failed to install the pre-commit hooks: %w", err)
	}

	e.Step("Pre-commit hooks installed")
	return nil
}

const (
	PreCommitHooksRev = "v6.0.0"
	DjangoUpgradeRev  = "1.33.0"
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

func (e *Env) generatePreCommitConfig(projectPath string) (string, error) {
	data := preCommitData{
		RuffEnabled: e.Cfg.SetupRuff,
		MypyEnabled: e.Cfg.SetupMypy,
		UVAvailable: e.toolchain == pyenv.UV,
		Templates:   e.Cfg.CreateTemplates,
		DjangoTrack: config.DjangoSeries(e.Cfg.DjangoVersion),
	}
	if !data.UVAvailable {
		versions, err := e.freezeVersions(projectPath)
		if err != nil {
			return "", err
		}
		hookTools := []string{PkgRuff, PkgDjLint, PkgPreCommit}
		var mypyDeps []string
		for _, spec := range append(append([]string{}, e.deps.runtime...), e.deps.dev...) {
			if !slices.Contains(hookTools, distributionName(spec)) {
				mypyDeps = append(mypyDeps, spec)
			}
		}
		if data.RuffDeps, err = exactPins([]string{PkgRuff}, versions, data.RuffEnabled); err != nil {
			return "", err
		}
		if data.DjLintDeps, err = exactPins([]string{PkgDjLint}, versions, data.Templates); err != nil {
			return "", err
		}
		if data.MypyDeps, err = exactPins(mypyDeps, versions, data.MypyEnabled); err != nil {
			return "", err
		}
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
	data.PreCommitHooksRev = PreCommitHooksRev
	data.DjangoUpgradeRev = DjangoUpgradeRev
	data.Language = "python"
	prefix := ""
	if data.UVAvailable {
		data.Language, prefix = "system", "uv run "
	}
	data.MypyEntry = prefix + "mypy ."
	data.RuffCheckEntry = prefix + "ruff check --fix"
	data.RuffFormatEntry = prefix + "ruff format"
	data.DjLintEntry = prefix + "djlint --lint --profile=django"
	return render("pre-commit-config.yaml.tmpl", data)
}
