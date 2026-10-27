# django-stack

[![Go Report Card](https://goreportcard.com/badge/github.com/vaibdix/django-stack)](https://goreportcard.com/report/github.com/vaibdix/django-stack)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A terminal UI that generates Django projects.

It finds the Python interpreters on your machine and offers only the Django
releases the chosen interpreter can run. Before it reports success, it runs the
new project's own checks: the Django system check and the test suite, plus Ruff
and mypy when you choose them.

## What you get

- Stacks: plain templates, HTMX, Tailwind v4, Tailwind + HTMX, or API only
- An optional API in DRF or Django Ninja, with a working CRUD example
- Add-ons: git, AllAuth, Ruff, mypy (strict, no explicit `Any`), pre-commit,
  split settings, pydantic settings, structlog, Sentry
- SQLite, or Postgres or MySQL in Docker
- A custom user model, VS Code config (`.vscode/`) and a start script
- Secure defaults: a Content-Security-Policy (native on Django 6.x, via
  `django-csp` on 5.2), hardened production settings and a `/healthz/` endpoint
- Locked, hashed dependencies: `uv.lock` with uv, `requirements*.txt` with pip

## Install

You need Go 1.27+ and Python 3.10+. [uv](https://github.com/astral-sh/uv) is
optional: without it the project is built with `venv` and pip.

```bash
go install github.com/vaibdix/django-stack/cmd/django-stack@latest
```

From source: `git clone https://github.com/vaibdix/django-stack.git && cd django-stack && make install`.

## Usage

```bash
django-stack                      # interactive setup
django-stack -config answers.json # no prompts; progress goes to stderr
django-stack -config answers.json -verbose  # also print each command's output as it runs
django-stack -config answers.json -json     # progress and output as one JSON event per line
django-stack -version
django-stack -install             # copy the binary to /usr/local/bin or ~/.local/bin
                                  # (%LOCALAPPDATA%\DjangoStack on Windows)
```

The setup asks for a project name, interpreter, Django version, database,
stack, add-ons and app name. `ctrl+c` cancels at any point.

Exit status: `0` created, `1` failed, `2` created but its own checks fail,
`130` cancelled, `143` terminated (SIGTERM). On `1`, the partial project and
its database container are removed; set `DJANGO_STACK_KEEP_FAILED=1` to keep
them. On `2`, the project is kept.

With uv, the project is built in a hidden folder beside the target
(`.my-project.django-stack-…`) and moved into place once its checks have run, so
a failed run never shows a half-built project at the target. With pip, the
project is built in place.

### Answers file

The keys are the setup's questions. An unknown key is an error. Comments and
trailing commas are allowed.

```json
{
  "name": "my-project",
  "python": "3.13",
  "django": "6.1.2",
  "database": "sqlite",
  "stack": "htmx",
  "app": "catalog",
  "addons": ["rest-framework", "ruff", "mypy", "pre-commit"]
}
```

| Key | Values |
|---|---|
| `python` | a version such as `3.13` (the newest installed patch is used), or an interpreter's path or name |
| `database` | `sqlite`, `postgres-docker`, `mysql-docker` |
| `stack` | `standard`, `htmx`, `tailwind`, `tailwind-htmx`, `api-only` |
| `addons` | `git`, `rest-framework`, `django-ninja`, `allauth`, `ruff`, `mypy`, `pre-commit`, `multi-env`, `pydantic`, `structlog`, `sentry` |
| `indexDate` | optional: a date (`2026-10-27`) or RFC 3339 time; packages are resolved as the index stood then |

Every run records the index date it resolved at in `django-stack.json`
(`answers.indexDate`), so generating again from that file resolves the same
versions. With pip, this needs a pip that supports `--uploaded-prior-to`; older
ones, such as pip 23 on Python 3.10, resolve against today's index and say so.
The manifest also carries a `schemaVersion`; older manifests are migrated when
read.

`pre-commit` needs `git`, so choosing it without `git` adds `git`. The setup
summary says so, and `django-stack.json` lists it under `resolved.addedAddOns`.

### Private package index

Django releases, and the hashes the pip path pins, come from the index your
installer uses: `UV_DEFAULT_INDEX` (or `UV_INDEX_URL`) with uv, `PIP_INDEX_URL`
with pip, otherwise PyPI. The index must serve the JSON Simple API (PEP 691).
An index set only in `pip.conf` or `uv.toml` is not read.

## DRF or Ninja

Pick one, not both: both use `<app>/api/` and `/api/v1/`, so the setup refuses
the pair. Each gives the same Book API (list, detail, create, update, delete,
`recent`), paginated and throttled, with a command that seeds sample data.

| | REST Framework | Django Ninja |
|---|---|---|
| Docs | `/api/schema/swagger-ui/` (drf-spectacular) | `/api/v1/docs` |
| Validation | serializers | pydantic schemas |
| Writes need | `IsAuthenticatedOrReadOnly` | `django_auth` (session) |

## Django versions

Only series that still get security fixes are offered. A series leaves the list
when its support ends.

| Django | Security fixes until | Python |
|---|---|---|
| 5.2 LTS | April 2028 | 3.10–3.14 |
| 6.0 | April 2027 | 3.12–3.14 |
| 6.1 | December 2027 | 3.12–3.14 |

A newer interpreter, such as 3.15, is listed but offers no Django release.

## Generated project

```
my-project/
├── .env / .env.example     # real values / the copy that is safe to commit
├── .env.*.example          # production and staging values, with split settings
├── docker-compose.yml      # Docker databases only
├── pyproject.toml          # dependencies, Ruff and mypy config
├── uv.lock                 # with uv; with pip: requirements.txt and requirements-dev.txt
├── django-stack.json       # its answers; pass it to -config to generate the project again
├── README.md / WELCOME.md  # how to run, test and deploy it / VS Code first steps
├── start-dev.sh / .bat     # opens VS Code; its folder-open task starts the servers
├── .pre-commit-config.yaml # with the pre-commit add-on
├── package.json, assets/   # Tailwind stacks
├── templates/ static/      # stacks with templates
├── my_project/             # settings.py, or settings/ when split
├── accounts/ core/         # custom user model, shared base models
└── my_app/                 # models, views, forms, api/, templates, tests, sample data
```

Settings read `.env` through django-environ, or through a generated pydantic
`conf.py` with the pydantic add-on. Either way, a missing or malformed value
stops startup.

## Databases

Postgres (`postgres:18`, `psycopg[binary]`) and MySQL (`mysql:9.7`,
`mysqlclient`) run from the project's `docker-compose.yml`, bound to
`127.0.0.1`, with per-project credentials in `.env`. `docker compose up -d
--wait db` starts the database again later. This needs Docker Engine 25+, which the
setup checks, and Compose 2.23.1+, which it does not check (current Docker
Desktop and OrbStack have both).

For MySQL, `DB_HOST` is `127.0.0.1`, not `localhost`: `mysqlclient` treats
`localhost` as a Unix socket and ignores the port.

## Development

```bash
make build    # binary into dist/
make install  # binary into GOBIN
make test     # go test ./...
make check    # the fast tier: fmt, vet, goldens and the offline tests (seconds)
make ci       # fmt, vet, comments, arch, deadcode, cross-build, goldens, race, e2e, lint, vulncheck
make content-check  # every configuration rendered and checked by py_compile, ruff, djlint and compose,
                    # then reference/ checked by ruff, mypy, djlint, makemigrations and its tests (minutes)
make reference      # rewrite reference/ after an intended output change
make live     # the live tier: network, Docker and end-to-end tests
make sweep RUN=<folder> [BASELINE=<run>]  # generate and check the pairwise project set in batches
make ui-baseline RUN=<folder>             # record the screenshots later sweeps compare with
make ci-act   # the CI workflow in containers, via act
make outdated # pins, actions, images and modules behind their latest release
make catalog-update  # rewrite the catalog's tool pins and Django support table from their sources
```

CI runs only locally: the workflow is `ci/workflows/ci.yml`, not under `.github/`, and
`make ci-act` runs it with [act](https://github.com/nektos/act). Releases are
cut by hand with [GoReleaser](https://goreleaser.com).

- The engine (`internal/scaffold`, `render`, `plan`, `catalog`, `config`,
  `graph`, `event`, `port`, `adapter`, `pyenv`, `dbdocker`, `command`, `assets`,
  `blueprint`, `pypi`, `releases`, `version`) must not depend on Bubble Tea, so
  it runs without a terminal. `make arch` enforces this.
- `internal/render` turns a `Spec` into files and `internal/plan` declares the
  run's effects; both are pure, and `make arch` keeps I/O out of them.
  `internal/scaffold` executes the plan through the ports in `internal/port`
  (real adapters in `internal/adapter`) and sends events that
  `internal/ui/text` and `internal/ui/tui` display.
- The rules (Django series, add-ons, stacks, databases) are data in
  `internal/catalog/catalog.toml`; `config.Parse` checks them and returns the
  `Spec` the engine runs on.
- Design decisions are recorded in [`docs/adr`](docs/adr).
- Generated output is pinned by golden files and by [`reference/`](reference),
  a complete project with every feature on. After an intended change, run
  `make goldens` and review the diff.
- `make live` also runs the tests that need the network, Docker or a real
  generated project. The full sweep, which generates a pairwise selection of
  174 projects and runs each one's own checks and browser screenshots, is in
  [`tools/matrix`](tools/matrix/README.md).

## License

MIT
