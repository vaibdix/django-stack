package scaffold

import (
	"testing"

	"github.com/vaibdix/django-stack/internal/config"
	"github.com/vaibdix/django-stack/internal/golden"
)

func TestGoldenProjectFiles(t *testing.T) {
	t.Parallel()

	drf := config.Project{
		ProjectName: "my-shop", AppName: "catalog", DjangoVersion: "6.1.2",
		SelectedPythonVersion: "3.13.13", DatabaseType: config.PostgresDocker,
	}
	drf.ApplyFeatures(append(config.StackHTMX.Features(),
		config.FeatureRestFramework, config.FeatureAllAuth, config.FeatureRuff, config.FeatureMypy,
		config.FeaturePreCommit, config.FeaturePydanticConfig, config.FeatureSentry))
	ninja := drf
	ninja.SetupRestFramework, ninja.SetupNinja = false, true

	withDRF := &Env{cfg: drf, node: pnpm}
	withNinja := &Env{cfg: ninja, node: pnpm}
	everyHook := preCommitData{RuffEnabled: true, MypyEnabled: true, Templates: true, DjangoTrack: "6.1"}
	everyHook.RuffDeps = []string{"ruff==0.17.1"}
	everyHook.DjLintDeps = []string{"djlint==1.46.6"}
	everyHook.MypyDeps = []string{"django-stubs[compatible-mypy]==6.1.2", "django==6.1.2", "mypy==2.4.0"}
	uvHooks, pipHooks := everyHook, everyHook
	uvHooks.UVAvailable = true

	cases := map[string]string{
		"files/accounts/admin.py":          generateAccountsAdmin(typingData{RuntimeGenerics: true}),
		"files/accounts/apps.py":           renderStatic("accounts_apps.py.tmpl"),
		"files/accounts/models.py":         render("accounts_models.py.tmpl", struct{ Stubs52 bool }{false}),
		"files/accounts/tests.py":          render("accounts_tests.py.tmpl", struct{ AllAuthPages, Ruff bool }{true, true}),
		"files/core/apps.py":               renderStatic("core_apps.py.tmpl"),
		"files/core/models.py":             renderStatic("core_models.py.tmpl"),
		"files/app/apps.py":                render("apps.py.tmpl", withDRF.appNames()),
		"files/app/fields.py":              generateBookFields(typingData{RuntimeGenerics: true}),
		"files/app/htmx.py":                generateHtmxSupport(),
		"files/app/tests/test_models.py":   render("tests_model.py.tmpl", withDRF.appNames()),
		"files/app/tests/test_pages.py":    render("tests_book.py.tmpl", withDRF.testsData()),
		"files/app/tests/test_api.py":      withDRF.apiTestsContent(),
		"files/app/tests/test_ninja.py":    withNinja.apiTestsContent(),
		"files/project/views.py":           render("project_views.py.tmpl", struct{ Templates bool }{true}),
		"files/project/conf.py":            render("conf.py.tmpl", withDRF.confTemplateData()),
		"files/project/gitignore":          renderStatic("gitignore.tmpl"),
		"files/project/README_uv.md":       withDRF.renderReadme(true),
		"files/project/README_pip.md":      withDRF.renderReadme(false),
		"files/config/pre-commit_uv.yaml":  renderPreCommitConfig(uvHooks),
		"files/config/pre-commit_pip.yaml": renderPreCommitConfig(pipHooks),
		"files/assets/tailwind.css":        renderStatic("tailwind-source.css.tmpl"),
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
