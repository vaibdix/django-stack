package render

import (
	"testing"

	"github.com/vaibdix/django-stack/internal/config"
	"github.com/vaibdix/django-stack/internal/golden"
)

func TestGoldenProjectFiles(t *testing.T) {
	t.Parallel()

	drf := config.Choices{Name: "my-shop", App: "catalog", Django: "6.1.2", Database: config.PostgresDocker}
	python := config.Resolved{PythonVersion: "3.13.13"}
	drf.Stack, drf.AddOns = config.StackHTMX, config.NewSet(config.RestFramework, config.AllAuth, config.Ruff, config.Mypy,
		config.PreCommit, config.Pydantic, config.Sentry)
	ninja := drf
	ninja.AddOns = drf.AddOns.Without(config.RestFramework).With(config.Ninja)

	withDRF := &Project{Cfg: mustSpec(drf), Resolved: python, Node: Pnpm}
	withNinja := &Project{Cfg: mustSpec(ninja), Resolved: python, Node: Pnpm}
	everyHook := PreCommitData{RuffEnabled: true, MypyEnabled: true, Templates: true, DjangoTrack: "6.1"}
	everyHook.RuffDeps = []string{"ruff==0.17.1"}
	everyHook.DjLintDeps = []string{"djlint==1.46.6"}
	everyHook.MypyDeps = []string{"django-stubs[compatible-mypy]==6.1.2", "django==6.1.2", "mypy==2.4.0"}
	uvHooks, pipHooks := everyHook, everyHook
	uvHooks.UVAvailable = true

	cases := map[string]string{
		"files/accounts/admin.py":          generateAccountsAdmin(typingData{RuntimeGenerics: true}),
		"files/accounts/apps.py":           RenderStatic("accounts_apps.py.tmpl"),
		"files/accounts/models.py":         Render("accounts_models.py.tmpl", struct{ Stubs52 bool }{false}),
		"files/accounts/tests.py":          Render("accounts_tests.py.tmpl", struct{ AllAuthPages, Ruff bool }{true, true}),
		"files/core/apps.py":               RenderStatic("core_apps.py.tmpl"),
		"files/core/models.py":             RenderStatic("core_models.py.tmpl"),
		"files/app/apps.py":                Render("apps.py.tmpl", withDRF.appNames()),
		"files/app/fields.py":              generateBookFields(typingData{RuntimeGenerics: true}),
		"files/app/htmx.py":                generateHtmxSupport(),
		"files/app/tests/test_models.py":   Render("tests_model.py.tmpl", withDRF.appNames()),
		"files/app/tests/test_pages.py":    Render("tests_book.py.tmpl", withDRF.testsData()),
		"files/app/tests/test_api.py":      withDRF.apiTestsContent(),
		"files/app/tests/test_ninja.py":    withNinja.apiTestsContent(),
		"files/project/views.py":           Render("project_views.py.tmpl", struct{ Templates bool }{true}),
		"files/project/conf.py":            Render("conf.py.tmpl", withDRF.confTemplateData()),
		"files/project/gitignore":          RenderStatic("gitignore.tmpl"),
		"files/project/README_uv.md":       withDRF.renderReadme(true),
		"files/project/README_pip.md":      withDRF.renderReadme(false),
		"files/config/pre-commit_uv.yaml":  RenderPreCommitConfig(uvHooks),
		"files/config/pre-commit_pip.yaml": RenderPreCommitConfig(pipHooks),
		"files/assets/tailwind.css":        RenderStatic("tailwind-source.css.tmpl"),
	}
	for _, env := range deployedEnvironments {
		cases["files/env/"+env.name+".example"] = withDRF.deployedEnvExample(env, "my_shop")
	}
	for name, content := range generateNinjaPackage("catalog", "my-shop") {
		cases["files/ninja/"+name] = content
	}

	for name, got := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			golden.Assert(t, name, got)
		})
	}
}
