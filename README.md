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

## Install

You need Go 1.26+ and Python 3.10+. [uv](https://github.com/astral-sh/uv) is
optional: without it the project is built with `venv` and pip.

```bash
go install github.com/vaibdix/django-stack/cmd/django-stack@latest
```

From source: `git clone https://github.com/vaibdix/django-stack.git && cd django-stack && make install`.

## Usage

```bash
django-stack                      # interactive setup
django-stack -config answers.json # no prompts; progress goes to stderr
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
├── docker-compose.yml      # Docker databases only
├── pyproject.toml          # dependencies, Ruff and mypy config
├── django-stack.jsonc      # its answers; pass it to -config to generate the project again
├── start-dev.sh / .bat     # opens VS Code; its folder-open task starts the servers
├── templates/ static/      # stacks with templates
├── my_project/             # settings.py, or settings/ when split
├── accounts/ core/         # custom user model, shared base models
└── my_app/                 # models, views, forms, api/, templates, tests
```

Settings read `.env` through django-environ, or through a generated pydantic
`conf.py` with the pydantic add-on. Either way, a missing or malformed value
stops startup.

## Databases

Postgres (`postgres:18`, `psycopg[binary]`) and MySQL (`mysql:8.4`,
`mysqlclient`) run from the project's `docker-compose.yml`, bound to
`127.0.0.1`, with per-project credentials in `.env`. `docker compose up -d
--wait db` starts the database again later. This needs Docker Engine 25+ and
Compose 2.23.1+ (current Docker Desktop and OrbStack have both).

For MySQL, `DB_HOST` is `127.0.0.1`, not `localhost`: `mysqlclient` treats
`localhost` as a Unix socket and ignores the port.

## Development

```bash
make build    # binary into dist/
make install  # binary into GOBIN
make test     # go test ./...
make ci       # fmt, vet, arch, goldens, race, e2e, lint, vulncheck
make ci-act   # the CI workflow in containers, via act
```

CI runs only locally: the workflows are in `ci/workflows/`, not `.github/`, and
`make ci-act` runs them with [act](https://github.com/nektos/act). Releases are
cut by hand with [GoReleaser](https://goreleaser.com).

- The engine (`internal/scaffold`, `config`, `assets`, `pypi`) must not depend
  on Bubble Tea, so it runs without a terminal. `make arch` enforces this.
- Generated output is pinned by golden files. After an intended change, run
  `make goldens` and review the diff.

## License

MIT
