package render

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/vaibdix/django-stack/internal/config"
)

func (p Project) ruffSection() string {
	metadata := p.PythonToolingMetadata()
	firstParty := []string{
		fmt.Sprintf(`"%s"`, p.Cfg.ModuleName()),
		fmt.Sprintf(`"%s"`, coreApp),
		fmt.Sprintf(`"%s"`, accountsApp),
	}
	if p.Cfg.App().Ok() {
		firstParty = append(firstParty, fmt.Sprintf(`"%s"`, p.Cfg.App().String()))
	}

	return Render("pyproject_ruff.toml.tmpl", struct{ Target, FirstParty, RuntimeEvaluated string }{metadata.RuffTargetVersion, strings.Join(firstParty, ", "), p.runtimeEvaluatedSection()})
}

func (p Project) runtimeEvaluated() (bases, decorators []string) {
	if p.usesPydanticModels() {
		bases = append(bases, "pydantic.BaseModel")
	}
	if p.Cfg.Has(config.Pydantic) {
		bases = append(bases, "pydantic_settings.BaseSettings")
	}
	if p.Cfg.Has(config.Ninja) {
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

func (p Project) runtimeEvaluatedSection() string {
	bases, decorators := p.runtimeEvaluated()
	if len(bases) == 0 && len(decorators) == 0 {
		return ""
	}
	data := struct{ Bases, Decorators string }{Bases: tomlStringArray(bases)}
	if len(decorators) > 0 {
		data.Decorators = tomlStringArray(decorators)
	}
	return Render("pyproject_ruff_runtime.toml.tmpl", data)
}

func (p Project) lazyAnnotations() bool {
	_, minor, ok := strings.Cut(p.PythonToolingMetadata().MypyPythonVersion, ".")
	if !ok {
		return false
	}
	number, err := strconv.Atoi(minor)
	return err == nil && number >= 14
}

func ruffFeature() feature {
	return feature{
		enabled: Has(config.Ruff),
		contribute: func(Project) contribution {
			return contribution{
				readme: func(d *readmeData) { d.Ruff = true },
				checks: []VerificationCheck{{Name: "ruff check", Args: []string{"-m", "ruff", "check", "."}}},
			}
		},
	}
}
