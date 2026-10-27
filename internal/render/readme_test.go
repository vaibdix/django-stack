package render

import (
	"strings"
	"testing"

	"github.com/vaibdix/django-stack/internal/config"
)

func TestTheReadmeDocumentsEverythingAnOperatorNeeds(t *testing.T) {
	t.Parallel()

	projectPath := t.TempDir()
	e := &Project{
		Cfg: mustSpec(config.Choices{
			Name:     "demo",
			App:      "catalog",
			Django:   "6.1",
			Database: config.PostgresDocker,
			AddOns:   config.NewSet(config.RestFramework, config.Ruff, config.Mypy, config.MultiEnv),
		}),
		Resolved: config.Resolved{PythonVersion: "Python 3.13.2"},
	}

	_ = projectPath
	readme := e.renderReadme(true)
	pipReadme := e.renderReadme(false)

	for _, fragment := range []string{
		"python -m venv .venv",
		".venv/bin/python -m pip install -r requirements-dev.txt",
		".venv/bin/python manage.py createsuperuser",
		`.venv\Scripts\python.exe`,
		".venv/bin/python -m ruff check .",
	} {
		if !strings.Contains(pipReadme, fragment) {
			t.Errorf("the pip README never mentions %q:\n%s", fragment, pipReadme)
		}
	}
	if strings.Contains(pipReadme, "uv ") {
		t.Errorf("a project built without uv tells its reader to use uv:\n%s", pipReadme)
	}

	for _, fragment := range []string{
		"uv sync",
		"docker compose up -d --wait db",
		"cp .env.example .env",
		"manage.py migrate",
		"uv run python manage.py createsuperuser",
		"uv run python manage.py test --settings=demo.settings.test",
		"uv run ruff check .",
		"uv run mypy .",
		"collectstatic",
		"gunicorn demo.wsgi:application",
		"/healthz/",
		"ENVIRONMENT",
	} {
		if !strings.Contains(readme, fragment) {
			t.Errorf("README.md never mentions %q, so that step is documented nowhere:\n%s", fragment, readme)
		}
	}
}

func TestReadmeDeploysInAnOrderThatWorks(t *testing.T) {
	t.Parallel()

	for _, uv := range []bool{true, false} {
		e := &Project{Cfg: mustSpec(config.Choices{Name: "my-shop", Database: config.PostgresDocker, AddOns: config.NewSet(config.MultiEnv)})}
		readme := e.renderReadme(uv)

		order := []string{"export DJANGO_SETTINGS_MODULE=", "manage.py migrate --noinput", "manage.py createcachetable", "manage.py collectstatic --noinput", "manage.py check --deploy", " gunicorn my_shop.wsgi:application"}
		last := -1
		for _, step := range order {
			at := strings.Index(readme, step)
			if at <= last {
				t.Errorf("uv=%v: deploy step %q is missing or out of order:\n%s", uv, step, readme)
			}
			last = at
		}
		if strings.Contains(readme, "\ngunicorn ") {
			t.Errorf("uv=%v: gunicorn is called bare, outside the project's environment:\n%s", uv, readme)
		}
		if strings.Index(readme, "cp .env.example .env") > strings.Index(readme, "docker compose up") {
			t.Errorf("uv=%v: compose reads DB_* from .env, so .env must exist before it starts:\n%s", uv, readme)
		}
	}
}

func TestReadmeDocumentsTheRoutesOfTheChosenAPI(t *testing.T) {
	t.Parallel()

	ninja := &Project{Cfg: mustSpec(projectForStack(config.StackAPIOnly, "catalog", config.Ninja))}
	readme := ninja.renderReadme(true)
	for _, drfOnly := range []string{"/api/schema/swagger-ui/", "/api-auth/login/"} {
		if strings.Contains(readme, drfOnly) {
			t.Errorf("a Ninja project's README lists the DRF route %s:\n%s", drfOnly, readme)
		}
	}
	if !strings.Contains(readme, "/api/v1/docs") {
		t.Errorf("a Ninja project's README never names its docs URL:\n%s", readme)
	}

	drf := &Project{Cfg: mustSpec(projectForStack(config.StackAPIOnly, "catalog", config.RestFramework))}
	if readme := drf.renderReadme(true); !strings.Contains(readme, "/api/schema/swagger-ui/") || strings.Contains(readme, "/api/v1/docs") {
		t.Errorf("a DRF project's README names the wrong docs URL:\n%s", readme)
	}
}

func TestTailwindReadmeExplainsHowToBuildTheStylesheet(t *testing.T) {
	t.Parallel()

	cfg := projectForStack(config.StackTailwind, "catalog")
	e := &Project{Cfg: mustSpec(cfg), Node: Pnpm}
	readme := e.renderReadme(true)
	for _, want := range []string{"Node.js and pnpm", "pnpm install", "pnpm run build:css", "pnpm run watch:css"} {
		if !strings.Contains(readme, want) {
			t.Errorf("a fresh clone cannot rebuild the CSS: the README never says %q:\n%s", want, readme)
		}
	}

	plain := (&Project{Cfg: mustSpec(projectForStack(config.StackStandard, "catalog"))}).renderReadme(true)
	if strings.Contains(plain, "Node.js") || strings.Contains(plain, "## Styles") {
		t.Errorf("a project without Tailwind is told to install Node:\n%s", plain)
	}
	if !strings.Contains(plain, "```\n\n## Environment") {
		t.Errorf("the Environment heading lost its blank line:\n%s", plain)
	}
}
