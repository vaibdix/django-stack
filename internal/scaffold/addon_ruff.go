package scaffold

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/vaibdix/django-stack/internal/config"
)

func (p project) ruffSection() string {
	metadata := p.pythonToolingMetadata()
	firstParty := []string{
		fmt.Sprintf(`"%s"`, p.cfg.ModuleName()),
		fmt.Sprintf(`"%s"`, coreApp),
		fmt.Sprintf(`"%s"`, accountsApp),
	}
	if p.cfg.App != "" {
		firstParty = append(firstParty, fmt.Sprintf(`"%s"`, p.cfg.App))
	}

	return render("pyproject_ruff.toml.tmpl", struct{ Target, FirstParty, RuntimeEvaluated string }{metadata.RuffTargetVersion, strings.Join(firstParty, ", "), p.runtimeEvaluatedSection()})
}

func (p project) runtimeEvaluated() (bases, decorators []string) {
	if p.usesPydanticModels() {
		bases = append(bases, "pydantic.BaseModel")
	}
	if p.cfg.Has(config.Pydantic) {
		bases = append(bases, "pydantic_settings.BaseSettings")
	}
	if p.cfg.Has(config.Ninja) {
		bases = append(bases, "ninja.Schema")
		for _, method := range []string{"get", "post", "put", "patch", "delete"} {
			decorators = append(decorators, "ninja.Router."+method)
		}
		decorators = append(decorators, "ninja.NinjaAPI.exception_handler")
	}
	return bases, decorators
}

func tomlStringArray(values []string) string {
	quoted := make([]string, 0, len(values))
	for _, value := range values {
		quoted = append(quoted, tomlString(value))
	}
	return "[" + strings.Join(quoted, ", ") + "]"
}

func (p project) runtimeEvaluatedSection() string {
	bases, decorators := p.runtimeEvaluated()
	if len(bases) == 0 && len(decorators) == 0 {
		return ""
	}
	section := `
[tool.ruff.lint.flake8-type-checking]
# These classes and decorators read annotations at runtime, so the imports the
# annotations name stay real imports rather than moving under "if TYPE_CHECKING:".
runtime-evaluated-base-classes = ` + tomlStringArray(bases) + "\n"
	if len(decorators) > 0 {
		section += "runtime-evaluated-decorators = " + tomlStringArray(decorators) + "\n"
	}
	return section
}

func (p project) lazyAnnotations() bool {
	_, minor, ok := strings.Cut(p.pythonToolingMetadata().MypyPythonVersion, ".")
	if !ok {
		return false
	}
	number, err := strconv.Atoi(minor)
	return err == nil && number >= 14
}

func ruffAddOn() addOn {
	return addOn{
		enabled: has(config.Ruff),
		contribute: func(project) contribution {
			return contribution{
				readme: func(d *readmeData) { d.Ruff = true },
				checks: []verificationCheck{{name: "ruff check", args: []string{"-m", "ruff", "check", "."}}},
			}
		},
	}
}
