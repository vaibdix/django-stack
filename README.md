# django-stack

[![Go Report Card](https://goreportcard.com/badge/github.com/vaibdix/django-cli)](https://goreportcard.com/report/github.com/vaibdix/django-cli)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A terminal UI for starting Django projects.

It finds the Python interpreters on your machine, lets you pick one, then only
offers the Django versions that interpreter can actually run. What you get out
is a project that already passes its own linter and type checker, so there's no
cleanup pass before your first commit.

## What you get

- An interpreter picker and a Django version list pulled live from PyPI
- A choice of stack: plain templates, HTMX, Tailwind v4, Tailwind + HTMX, or API only
- An API layer if you want one, either DRF or Django Ninja, with a working CRUD example
- AllAuth, Postgres or MySQL in Docker, git init, a VS Code workspace
- Ruff, Mypy, pre-commit, split settings, pydantic settings, structlog, Sentry

Everything generated is typed. `mypy --strict` passes with `Any` disallowed, and
each project gets a `ruff` and `djlint` pass before setup reports success.

## Install

You'll need Go 1.26 or newer and Python 3.10 or newer. Which Django versions show
up depends on the interpreter you choose. Installing
[uv](https://github.com/astral-sh/uv) is optional but makes creating the virtualenv
about ten times faster.

```bash
go install github.com/vaibdix/django-cli/cmd/django-stack@latest
```

From source:

```bash
git clone https://github.com/vaibdix/django-cli.git
cd django-cli && make install
```

## Using it

```bash
django-stack            # this is the one you want
django-stack --help
django-stack --version
django-stack --install  # drops the binary somewhere on PATH
```

It'll ask for a project name, an interpreter, a Django version, a database, a
stack, whatever add-ons you want, and an app name. `ctrl+c` gets you out at any
point.

If a run fails, the half-built project and its database container are removed,
so you can retry with the same name. Set `DJANGO_STACK_KEEP_FAILED=1` to keep
them for inspection instead. A project that builds but fails its own checks is
always kept.

The exit status says how it went, and the error is printed again after the
interface closes: `0` created, `1` failed (the partial project was removed), `2`
created but its own checks fail (kept), `130` cancelled.

### Without the interactive setup

`django-stack -config answers.json` asks nothing: it reads the answers from a
JSON file and prints progress to stderr. The keys are the questions the setup
asks, and an unknown key is an error rather than ignored:

```json
{
  "name": "my-project",
  "python": "/opt/homebrew/bin/python3.13",
  "django": "6.1.2",
  "database": "sqlite",
  "stack": "htmx",
  "app": "catalog",
  "addons": ["rest-framework", "ruff", "mypy", "pre-commit"]
}
```

`database` is `sqlite`, `postgres-docker` or `mysql-docker`; `stack` is
`standard`, `htmx`, `tailwind`, `tailwind-htmx` or `api-only`; `addons` uses the
names `django-stack.jsonc` records (`git`, `rest-framework`, `django-ninja`,
`allauth`, `ruff`, `mypy`, `pre-commit`, `multi-env`, `pydantic`, `structlog`,
`sentry`). The same rules apply as in the setup, and the exit status is the same.

### A private package index

The Django releases on offer, and the hashes the pip path pins, come from the
index the installer uses: `UV_DEFAULT_INDEX` (or the older `UV_INDEX_URL`) with
uv, `PIP_INDEX_URL` with pip, and PyPI when neither is set. The index must serve
the JSON Simple API (PEP 691); one that serves only HTML is reported as an
error. An index configured only in `pip.conf` or `uv.toml` is not read, so set
the variable to use it.

## Stacks and add-ons

The stack decides what HTML the project is built from. Pick one of Standard,
Standard + HTMX, Standard + Tailwind, Standard + Tailwind + HTMX, or API only if
you don't want templates at all.

Add-ons are a multi-select, and they're mostly independent: git init, REST
Framework, Django Ninja, AllAuth, Ruff, Mypy, pre-commit, split settings, typed
settings, structlog, Sentry.

### DRF or Ninja?

Pick one. Both write to `<app>/api/` and both mount `/api/v1/`, so choosing both
would just have them fight over the same routes.

| | REST Framework | Django Ninja |
|---|---|---|
| Docs at | `/api/schema/swagger-ui/` via drf-spectacular | `/api/v1/docs`, built in |
| Validation | serializers, plus pydantic if typed settings are on | pydantic all the way down |
| Writes | `IsAuthenticatedOrReadOnly` | `django_auth`, session based |
| Works with | every Django version here | every Django version here |

You get the same Book example either way: list, detail, create, update, delete
and a `recent` endpoint, paginated and throttled, plus a command to seed it.

## Django versions

Only releases that still get security fixes are offered:

| Django | Security fixes until | 3.10 | 3.11 | 3.12 | 3.13 | 3.14 |
|--------|----------------------|:----:|:----:|:----:|:----:|:----:|
| 5.2 LTS | April 2028          | ✓    | ✓    | ✓    | ✓    | ✓    |
| 6.0    | April 2027           |      |      | ✓    | ✓    | ✓    |
| 6.1    | December 2027        |      |      | ✓    | ✓    | ✓    |

A series drops out of the list on its own once its support ends. No Django
release supports Python 3.15 yet, so a 3.15 interpreter is listed but offers no
Django version until one does.

## What the project looks like

```
my-project/
├── .env / .env.example         # real secrets, and the safe copy to commit
├── docker-compose.yml          # only if you picked a Docker database
├── pyproject.toml              # ruff and mypy config
├── django-stack.jsonc          # a record of what this was generated from
├── start-dev.sh                # opens VS Code, whose folder-open task starts the servers
├── templates/ static/          # base, index, error pages
├── my_project/                 # settings.py, or settings/ if you split them
├── accounts/ core/             # custom user model, shared base models
└── my_app/                     # models, views, forms, api/, templates
```

Config comes from `.env` through `django-environ`. Turn on typed settings and
you get a generated `conf.py` instead, which validates everything at startup.
`.env.example` has the same keys with the secrets taken out, so it's safe to
commit.

## Databases

SQLite is the default and needs nothing from you. Postgres and MySQL run in
Docker containers bound to `127.0.0.1`, with per-project credentials written
into `.env`. The container is started from the project's own
`docker-compose.yml`, so `docker compose up -d --wait db` brings back the same
database later. This needs Docker Engine 25 or newer with Compose v2.23 or newer,
which current Docker Desktop and OrbStack releases include.

| Database | Image | Driver |
|---|---|---|
| Postgres | `postgres:18` | `psycopg[binary]` |
| MySQL | `mysql:8.4` | `mysqlclient` |

`DB_HOST` is set to `127.0.0.1` and not `localhost`, which looks fussy until you
hit it: `mysqlclient` treats `localhost` as a Unix socket path and quietly
ignores the port.

## Working on it

```bash
make build      # binary into dist/
make install    # binary into GOBIN
make test       # go test ./...
make ci         # fmt, vet, arch, goldens, race, e2e, lint
make ci-act     # the same thing in containers, via act
```

CI is local only. The workflows are versioned in `ci/workflows/`, outside
`.github/`, so nothing runs on GitHub; `make ci-act` feeds them to
[act](https://github.com/nektos/act). Releases are cut by hand with
[GoReleaser](https://goreleaser.com).

Two things to know before you change the engine. Only `internal/tui` is allowed
to import Bubble Tea, because everything underneath has to run without a
terminal for the tests to work at all, and `make arch` will fail if that slips.
And generated output is pinned by golden files, so run `make goldens` after any
deliberate change and actually read the diff. A golden diff nobody can explain
is a bug.

## License

MIT
