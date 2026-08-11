# django-stack

[![Go Report Card](https://goreportcard.com/badge/github.com/vaibdix/django-cli)](https://goreportcard.com/report/github.com/vaibdix/django-cli)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

`django-stack` is a Bubble Tea-powered TUI for creating Django projects with a guided setup flow. It helps you choose a local Python interpreter, picks a compatible stable Django release, and scaffolds a ready-to-run project with optional Tailwind CSS v4, Django REST Framework (with example CRUD app), PostgreSQL/MySQL via Docker, Git init, VS Code dev tooling, and developer tools like Ruff, Mypy, pre-commit hooks, and multi-environment settings — inspired by [cookiecutter-django](https://github.com/cookiecutter/cookiecutter-django).

## Features

- Bubble Tea + Huh interactive TUI
- Automatic virtual environment setup
- Python interpreter picker seeded from your machine (with pyenv auto-detection and fix)
- Django version picker showing the latest compatible stable releases
- Built-in Tailwind CSS v4 integration
- Django REST Framework setup with example Book model CRUD + serializers
- HTMX integration for modern interactions without writing JavaScript
- Django AllAuth for authentication (login, signup, social auth)
- PostgreSQL and MySQL support via Docker with auto-generated `docker-compose.yml` (with health checks)
- **Environment variables management** (`django-environ`, `python-dotenv`)
- **Production-ready security** — Argon2 password hashing, security headers, HSTS, secure cookies
- **Automatic scaffold formatting** for Python and Django templates on every generated project
- **Ruff linting & formatting** with Django-specific rules when selected
- **Mypy type checking** with `django-stubs` (and `djangorestframework-stubs` if DRF selected)
- **Pre-commit hooks** — auto-lint on commit with ruff, djLint, django-upgrade
- **Multi-env settings** — split `settings.py` into `base.py`, `local.py`, `production.py`, `test.py`
- VS Code workspace auto-start (`tasks.json`, `launch.json`, `extensions.json`, `settings.json`)
- `start-dev.sh` launcher script to open VS Code and start dev servers
- `django-browser-reload` for live page updates during development
- Git repository initialization with `.gitignore`
- Cross-platform support (Windows, macOS, Linux)
- Bundled fallback Django release list when PyPI is unavailable

## Performance

`django-stack` uses [`uv`](https://github.com/astral-sh/uv) for faster virtualenv creation when it is available, and falls back to the selected Python interpreter otherwise:

| Operation | With `pip` | With `uv` | Improvement |
|-----------|------------|-----------|-------------|
| Virtual environment creation | 5-10s | 0.5-1s | **10x faster** |
| Django installation | 15-30s | 2-5s | **10x faster** |
| Total setup time | 30-60s | 3-10s | **Up to 20x faster** |

When `uv` is available, `django-stack` still installs packages into the project-local `.venv`. It does not install Django or project dependencies into your system Python.

## Installation

### Prerequisites

- **Go 1.26** or higher
- **Python 3.8** or higher
- **uv** (optional, but recommended)

### Install uv for Better Performance

```bash
pip install uv
```

### Install with `go install`

```bash
go install github.com/vaibdix/django-cli/cmd/django-stack@latest
```

### Building from Source

```bash
git clone https://github.com/vaibdix/django-cli.git
cd django-cli
make build
```

## Usage

### Quick Start

```bash
django-stack
```

### CLI Flags

```bash
django-stack --help
django-stack --version
django-stack --install
```

### Setup Flow

The default invocation launches the TUI. The interactive setup flow is:

1. **Project name** — folder name for the project (hyphens allowed, defaults to `django-stack`)
2. **Python interpreter** — discovered from your system, with the default preselected
3. **Django version** — filtered to the latest stable releases compatible with the selected Python
4. **Database type** — SQLite (default), PostgreSQL (Docker), or MySQL (Docker)
5. **Project features** — multi-select: Standard Django Project (templates & static), Git init, Tailwind CSS v4, Django REST Framework, HTMX, Django AllAuth, Ruff, Mypy, Pre-commit Hooks, Multi-env Settings
6. **App name** — prompted when Templates, REST Framework, HTMX, or AllAuth is selected
7. **Setup** — progress bar while the project is scaffolded
8. **Dev server prompt** — option to open VS Code and start development immediately

## Django Compatibility

Django versions are filtered based on the selected Python interpreter:

| Django | Python 3.8 | Python 3.10 | Python 3.12 | Python 3.13 | Python 3.14 |
|--------|:----------:|:-----------:|:-----------:|:-----------:|:-----------:|
| 4.2    | ✓          | ✓           | ✓           |             |             |
| 5.0    |            | ✓           | ✓           |             |             |
| 5.1    |            | ✓           | ✓           | ✓           |             |
| 5.2    |            | ✓           | ✓           | ✓           | ✓           |
| 6.0    |            |             | ✓           | ✓           | ✓           |

## Generated Project Structure

A fully-featured generated project includes:

```
my-project/
├── .env                           # Environment variables (NEVER commit)
├── .env.example                   # Template for version control
├── .gitignore                     # When git init is enabled
├── .vscode/
│   ├── extensions.json            # Recommended VS Code extensions
│   ├── launch.json                # Debug configurations
│   ├── settings.json              # Workspace settings
│   └── tasks.json                 # Auto-start dev tasks on folder open
├── .pre-commit-config.yaml        # Pre-commit hooks (when pre-commit enabled)
├── docker-compose.yml             # PostgreSQL container (when Docker DB selected)
├── manage.py
├── package.json                   # Tailwind CSS build scripts (when Tailwind enabled)
├── pyproject.toml                 # Ruff/Mypy config (when dev tools enabled)
├── start-dev.sh                   # Opens VS Code + starts dev servers
├── WELCOME.md                     # Getting started guide
├── templates/
│   ├── base.html                  # Base template with Tailwind/static setup
│   ├── index.html                 # Landing page
│   └── api-docs.html              # REST API documentation page
├── static/
│   ├── src/styles.css             # Tailwind source CSS
│   ├── dist/styles.css            # Compiled CSS output
│   └── js/main.js                 # JavaScript entry point
├── my_project/                    # Django project module (hyphens → underscores)
│   ├── __init__.py
│   ├── settings.py                # Configured with environ, security, etc.
│   ├── settings/                  # When multi-env enabled (replaces settings.py)
│   │   ├── __init__.py
│   │   ├── base.py                #   Shared config (apps, middleware, DB, etc.)
│   │   ├── local.py               #   DEBUG=True, console email, debug toolbar hints
│   │   ├── production.py          #   DEBUG=False, SSL, HSTS, secure cookies
│   │   └── test.py                #   SQLite, MD5 hasher, in-memory email
│   ├── urls.py
│   ├── views.py                   # Landing page view
│   ├── context_processors.py      # Project name context processor
│   ├── wsgi.py
│   └── asgi.py
├── app_name/                      # Django app (when Templates or REST selected)
│   ├── __init__.py
│   ├── admin.py
│   ├── apps.py
│   ├── forms.py                   # Book model forms
│   ├── models.py                  # Book model
│   ├── serializers.py             # DRF serializers (when REST selected)
│   ├── api.py                     # DRF viewsets (when REST selected)
│   ├── views.py                   # Template-based CRUD views
│   ├── urls.py
│   ├── tests.py
│   ├── templates/app_name/        # App-specific templates
│   │   ├── book_list.html
│   │   ├── book_form.html
│   │   └── book_delete.html
│   ├── migrations/
│   │   └── 0001_initial.py
│   └── management/
│       └── commands/
│           └── create_sample_data.py
└── .venv/                         # Virtual environment
```

## Environment Variables

Django Stack configures environment variables using [`django-environ`](https://django-environ.readthedocs.io/) and [`python-dotenv`](https://github.com/theskumar/python-dotenv).

### Files Created

| File | Purpose |
|------|---------|
| `.env` | Your actual environment variables (never commit this!) |
| `.env.example` | Template showing available variables |

### SQLite Configuration

```bash
# Django Settings
SECRET_KEY=<auto-generated>
DEBUG=True
ALLOWED_HOSTS=localhost,127.0.0.1
```

### PostgreSQL (Docker) Configuration

```bash
# Django Settings
SECRET_KEY=<auto-generated>
DEBUG=True
ALLOWED_HOSTS=localhost,127.0.0.1

# Database (PostgreSQL Docker)
DB_NAME=my_project
DB_USER=postgres
DB_PASSWORD=<auto-generated-hex>
DB_HOST=localhost
DB_PORT=5432
```

### MySQL (Docker) Configuration

```bash
# Django Settings
SECRET_KEY=<auto-generated>
DEBUG=True
ALLOWED_HOSTS=localhost,127.0.0.1

# Database (MySQL Docker)
DB_NAME=my_project
DB_USER=django_user
DB_PASSWORD=<auto-generated-hex>
DB_ROOT_PASSWORD=<auto-generated-hex>
DB_HOST=localhost
DB_PORT=3306
```

The database engine is hardcoded in `settings.py` based on the database type selected during setup — there is no `DB_ENGINE` variable.

### Best Practices

1. **Never commit `.env`** — it's added to `.gitignore` automatically
2. **Keep `.env.example`** — share this with your team (no secrets)
3. **Use strong SECRET_KEYs** — generate new keys for production
4. **Use different configs per environment** — dev, staging, production

## Security Features

Django Stack generates projects with production-ready security settings inspired by [cookiecutter-django](https://github.com/cookiecutter/cookiecutter-django):

### Password Security

- **Argon2 password hasher** (primary) with PBKDF2, bcrypt, and scrypt fallbacks
- 4 password validators: similarity, minimum length, common password, and numeric checks
- `argon2-cffi` is installed automatically alongside Django

### Security Headers & Cookies

Always-on (safe for development and production):
- `SECURE_CONTENT_TYPE_NOSNIFF = True`
- `SESSION_COOKIE_HTTPONLY = True`
- `CSRF_COOKIE_HTTPONLY = True`
- `X_FRAME_OPTIONS = 'DENY'`

Production-only (behind `if not DEBUG:`):
- `SECURE_SSL_REDIRECT` — redirects HTTP to HTTPS
- `SECURE_HSTS_SECONDS` — 1 year HSTS with preload
- `SECURE_PROXY_SSL_HEADER` — trust `X-Forwarded-Proto`
- `SESSION_COOKIE_SECURE` / `CSRF_COOKIE_SECURE` — HTTPS-only cookies

### Docker Health Checks

Both PostgreSQL and MySQL `docker-compose.yml` files include health checks:
- **PostgreSQL**: `pg_isready -U ${DB_USER}` with 10s interval, 5 retries
- **MySQL**: `mysqladmin ping -h localhost` with 10s interval, 5 retries

### Production Environment Hints

The generated `.env.example` includes commented-out production security variables (`SECURE_SSL_REDIRECT`, `SECURE_HSTS_SECONDS`) as reminders for deployment.

## Developer Tools (Optional)

Django Stack can optionally set up developer tooling during project creation. Select any combination from the TUI multi-select.

| Tool | Pip Packages | Config File |
|------|-------------|-------------|
| **Ruff (Linter + Formatter)** | `ruff`, `djlint` | `pyproject.toml` |
| **Mypy (Type Checking)** | `mypy`, `django-stubs`, `djangorestframework-stubs`* | `pyproject.toml` |
| **Pre-commit Hooks** | `pre-commit` | `.pre-commit-config.yaml` |
| **Multi-env Settings** | — | `settings/` directory |

*`djangorestframework-stubs` is only installed when Django REST Framework is also selected.

### Ruff

Every scaffold already runs a one-time formatting pass with `ruff` and `djlint` before setup completes. Selecting **Ruff** adds persistent Ruff configuration to `pyproject.toml` with Django-specific lint rules (`DJ`, `S`, `SIM`, `UP`, etc.), 120-char line length, and isort configuration.

### Mypy

Installs `mypy` with `django-stubs` (and `djangorestframework-stubs` if DRF is selected). Writes `[tool.mypy]` and `[tool.django-stubs]` sections to `pyproject.toml`. Automatically sets `django_settings_module` to `<module>.settings.local` when multi-env is also enabled.

### Pre-commit Hooks

Installs `pre-commit` and generates `.pre-commit-config.yaml` with:
- **pre-commit-hooks** — trailing whitespace, end-of-file fixer, YAML check, debug statements
- **ruff-pre-commit** — ruff lint + format (only when Ruff is also selected)
- **djLint** — Django template linting
- **django-upgrade** — automatic Django deprecation fixes

Runs `pre-commit install` to activate hooks. Automatically enables Git init if not already selected. Pre-commit is for ongoing hook-based validation; it no longer controls whether the initial scaffold formatting pass happens.

### Mandatory Scaffold Formatting

Every generated project gets a required post-generation cleanup pass before setup is reported as successful:

- `ruff check --fix .`
- `ruff format .`
- `djlint --reformat` on generated Django templates

This cleanup always runs, even when neither **Ruff** nor **Pre-commit Hooks** is selected. If formatting fails, project generation stops and returns the formatter error instead of leaving a noisy scaffold behind.

### Multi-env Settings

Splits the completed `settings.py` into a `settings/` directory:

| File | `DEBUG` | `DJANGO_SETTINGS_MODULE` default |
|------|---------|----------------------------------|
| `base.py` | — | — |
| `local.py` | `True` | `manage.py` |
| `production.py` | `False` | `wsgi.py`, `asgi.py` |
| `test.py` | `False` | — |

- `base.py` contains all shared configuration (apps, middleware, database, templates, etc.)
- `local.py` adds console email backend and Django Debug Toolbar hints
- `production.py` adds SSL redirect, HSTS, secure cookies
- `test.py` uses SQLite, MD5 password hasher, and in-memory email for speed

## Docker / PostgreSQL

When you select **PostgreSQL (Docker)** during setup, django-stack generates a `docker-compose.yml` with a PostgreSQL 16 container:

```yaml
services:
  db:
    image: postgres:16
    restart: unless-stopped
    env_file:
      - .env
    environment:
      POSTGRES_DB: ${DB_NAME}
      POSTGRES_USER: ${DB_USER}
      POSTGRES_PASSWORD: ${DB_PASSWORD}
    ports:
      - "${DB_PORT}:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ${DB_USER}"]
      interval: 10s
      timeout: 5s
      retries: 5

volumes:
  postgres_data:
```

Start the database before running Django:

```bash
docker compose up -d
python manage.py migrate
python manage.py runserver
```

The port and password are auto-generated and written to `.env`.

### MySQL (Docker)

When you select **MySQL (Docker)** during setup, django-stack generates a `docker-compose.yml` with a MySQL 8.0 container:

```yaml
services:
  db:
    image: mysql:8.0
    restart: unless-stopped
    env_file:
      - .env
    environment:
      MYSQL_DATABASE: ${DB_NAME}
      MYSQL_USER: ${DB_USER}
      MYSQL_PASSWORD: ${DB_PASSWORD}
      MYSQL_ROOT_PASSWORD: ${DB_ROOT_PASSWORD}
    ports:
      - "${DB_PORT}:3306"
    volumes:
      - mysql_data:/var/lib/mysql
    command: --default-authentication-plugin=mysql_native_password
    healthcheck:
      test: ["CMD", "mysqladmin", "ping", "-h", "localhost"]
      interval: 10s
      timeout: 5s
      retries: 5

volumes:
  mysql_data:
```

Start the database before running Django:

```bash
docker compose up -d
python manage.py migrate
python manage.py runserver
```

The port and passwords are auto-generated and written to `.env`. The `mysqlclient` pip package is installed automatically during project setup.

## Development

### Source Structure

```
django-stack/
├── cmd/cli/                    # Entry point
│   ├── main.go
│   ├── install_unix.go
│   └── install_windows.go
├── internal/
│   ├── ui/                     # TUI components and project scaffolding
│   │   ├── model.go            #   Bubble Tea model, ProjectConfig & initialization
│   │   ├── view.go             #   TUI rendering
│   │   ├── update.go           #   TUI message handling
│   │   ├── types.go            #   Step enum & message types
│   │   ├── constants.go        #   File-name & path constants
│   │   ├── settings.go         #   Django settings.py generator (12 methods)
│   │   ├── creator.go          #   Project creation orchestrator
│   │   ├── python.go           #   Python discovery & path helpers
│   │   ├── python_options.go   #   Interpreter picker logic
│   │   ├── venv.go             #   Virtual-env creation (uv / pip)
│   │   ├── django.go           #   Django admin startproject wrapper
│   │   ├── django_versions.go  #   Version compatibility matrix
│   │   ├── packages.go         #   Pip/uv install helpers (installAndReport)
│   │   ├── api.go              #   REST Framework orchestrator
│   │   ├── api_generators.go   #   REST file generators (models, serializers, …)
│   │   ├── api_templates.go    #   REST template generators
│   │   ├── templates_ui.go     #   HTML template & URL scaffolding
│   │   ├── app.go              #   Django app creation
│   │   ├── allauth.go          #   django-allauth integration
│   │   ├── htmx.go             #   HTMX support
│   │   ├── ruff.go             #   Ruff linter + formatter setup
│   │   ├── mypy.go             #   Mypy type checking setup
│   │   ├── precommit.go        #   Pre-commit hooks setup
│   │   ├── multienv.go         #   Multi-env settings split
│   │   ├── pyproject.go        #   pyproject.toml editor
│   │   ├── tailwind.go         #   Tailwind CSS v4 (npm)
│   │   ├── docker.go           #   docker-compose.yml generation
│   │   ├── postgres.go         #   PostgreSQL configuration
│   │   ├── env.go              #   .env / .env.example generation
│   │   ├── migrations.go       #   Migrations, admin user, sample data
│   │   ├── scripts.go          #   start-dev.sh generation
│   │   ├── vscode.go           #   VS Code workspace files
│   │   ├── styles.go           #   CSS generation
│   │   ├── startup.go          #   Post-setup dev-server launch
│   │   ├── utils.go            #   Shared helpers
│   │   ├── settings_test.go    #   Settings unit tests
│   │   ├── api_generators_test.go # API generator unit tests
│   │   └── test_exports.go     #   Test-only exported helpers
│   ├── templates/              # HTML template content
│   │   ├── content.go          #   Dispatcher (TemplateContent)
│   │   ├── nav.go              #   Navigation bar (authNavLinks)
│   │   ├── base_templates.go   #   Base HTML layout
│   │   ├── index_templates.go  #   Landing page
│   │   ├── allauth_templates.go#   Auth page templates
│   │   ├── api_docs_templates.go#  REST API docs page
│   │   └── postgres_readme.go  #   PostgreSQL WELCOME.md content
│   ├── template/               # Template loader
│   ├── errors/                 # Error types
│   └── logger/                 # Logging
├── test/ui/                    # Integration tests
│   └── ui_test.go
├── scripts/                    # Manual smoke helpers
├── dist/                       # Build output
├── .github/workflows/          # CI (test + release)
├── makefile
├── go.mod
└── go.sum
```

### Build

```bash
make build      # build for the current platform into dist/
make install    # build and install into $GOBIN
make test       # run the test suite
make ci         # everything CI runs: fmt-check, vet, test -race, lint
make lint       # golangci-lint
make clean      # remove build and coverage artifacts
make dev        # hot-reload build (requires air)
```

Release binaries are produced by [GoReleaser](https://goreleaser.com) from
`.goreleaser.yaml` when a tag is pushed — there is no cross-compile target to
run by hand, and built binaries are never committed.

See [CONTRIBUTING.md](CONTRIBUTING.md) for the architecture rules, the
golden-file workflow, and the naming conventions.

### Test

```bash
go test ./...
# or
make test
```

## License

MIT License
