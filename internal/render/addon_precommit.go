package render

import (
	"fmt"
	"slices"
	"strings"

	"github.com/vaibdix/django-stack/internal/config"
)

func preCommitHooksRev() string { return config.Tools().PreCommitHooks }

func djangoUpgradeRev() string { return config.Tools().DjangoUpgrade }

type PreCommitData struct {
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

func ExactPins(specs []string, versions map[string]string, wanted bool) ([]string, error) {
	if !wanted {
		return nil, nil
	}
	pins := make([]string, 0, len(specs))
	for _, spec := range specs {
		version, ok := installedVersion(spec, versions)
		if !ok {
			return nil, fmt.Errorf("%s was installed but the package freeze does not list it", DistributionName(spec))
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

func RenderPreCommitConfig(data PreCommitData) string {
	data.PreCommitHooksRev = preCommitHooksRev()
	data.DjangoUpgradeRev = djangoUpgradeRev()
	data.Language = "python"
	data.MypyEntry = "mypy ."
	data.RuffCheckEntry = "ruff check --fix"
	data.RuffFormatEntry = "ruff format"
	data.DjLintEntry = "djlint --lint --profile=django"
	return Render("pre-commit-config.yaml.tmpl", data)
}

func preCommitFeature() feature {
	return feature{
		enabled: Has(config.PreCommit),
		contribute: func(Project) contribution {
			return contribution{
				dev:    []string{PkgPreCommit},
				readme: func(d *readmeData) { d.PreCommit = true },
				checks: []VerificationCheck{{
					Name: "pre-commit config",
					Args: []string{"-m", "pre_commit", "validate-config", PreCommitConfigFile},
				}},
			}
		},
	}
}
