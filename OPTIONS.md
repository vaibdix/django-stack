# django-stack CLI Options Reference

Complete reference for all CLI flags, database options, feature toggles, and their combinations.

---

## Section 1: CLI Flags

| Flag | Alias | Description |
|------|-------|-------------|
| `--help` | `-h` | Show usage information and exit |
| `--version` | `-v` | Show build version, commit hash, and build date |
| `--install` | — | Install the binary into your local PATH |
| *(none)* | — | Launch the interactive Bubble Tea TUI setup |

---

## Section 2: Database Options

| Option | Internal Value | Docker Image | Pip Package | Extra `.env` Vars |
|--------|---------------|--------------|-------------|-------------------|
| SQLite (default) | `sqlite` | — | — | — |
| PostgreSQL (Docker) | `postgres-docker` | `postgres:16` | `psycopg2-binary` | `DB_NAME`, `DB_USER`, `DB_PASSWORD`, `DB_HOST`, `DB_PORT` |
| MySQL (Docker) | `mysql-docker` | `mysql:8.4` | `mysqlclient` | `DB_NAME`, `DB_USER`, `DB_PASSWORD`, `DB_ROOT_PASSWORD`, `DB_HOST`, `DB_PORT` |

**Common `.env` vars** (all databases): `SECRET_KEY`, `DEBUG`, `ALLOWED_HOSTS`

**Generated files by database:**

| File | SQLite | PostgreSQL (Docker) | MySQL (Docker) |
|------|--------|---------------------|----------------|
| `.env` | Yes | Yes | Yes |
| `.env.example` | Yes | Yes | Yes |
| `docker-compose.yml` | — | Yes | Yes |

---

## Section 3: Stack

The stack is a single choice and decides which template variant the project is generated from.
Every stack except `API only` sets `CreateTemplates`, which is what `base.html` and the root
URL depend on.

| Stack | Flags set | Template variant |
|-------|-----------|------------------|
| Standard Django | `CreateTemplates` | `plain` |
| Standard + HTMX | `CreateTemplates`, `SetupHTMX` | `htmx` |
| Standard + Tailwind CSS v4 | `CreateTemplates`, `SetupTailwind` | `tailwind` |
| Standard + Tailwind + HTMX | `CreateTemplates`, `SetupTailwind`, `SetupHTMX` | `tailwind_htmx` |
| API only | none | — (root URL redirects to the schema UI, or the admin without REST) |

## Section 4: Add-on Options

Every project installs these **base packages**: `django-environ` — unless **Typed Python** is
selected, which replaces it with `pydantic` and `pydantic-settings`.

| # | Feature | Config Flag | Pip Packages | Django Apps (`INSTALLED_APPS`) | Middleware | Key Generated Files |
|---|---------|-------------|--------------|-------------------------------|------------|---------------------|
| G | Initialize Git | `InitializeGit` | — | — | — | `.gitignore` |
| R | Django REST Framework | `SetupRestFramework` | `djangorestframework` | `rest_framework` | — | `serializers.py`, `api.py`, `models.py` (Book), `forms.py`, `api-docs.html` |
| A | Django AllAuth | `SetupAllAuth` | `django-allauth` | `django.contrib.sites`, `allauth`, `allauth.account` | `AccountMiddleware` | Auth templates (login, signup, logout) |
| P | Typed Python | `SetupPydanticConf` | `pydantic`, `pydantic-settings` | — | — | `<module>/conf.py`, `<app>/schemas.py` |
| L | Structured Logging | `SetupStructlog` | `structlog`, `django-structlog` | `django_structlog` | `RequestMiddleware` | `LOGGING` + `structlog.configure()` in settings |
| E | Error Tracking | `SetupSentry` | `sentry-sdk` | — | — | Guarded `sentry_sdk.init()` in settings, `SENTRY_*` in `.env.example` |

**App name prompt:** The TUI prompts for an app name when any of **S**, **R**, **H**, or **A** is selected. Git (G) and Tailwind (T) alone do not trigger the app name prompt.

---

## Section 5: Feature Combinations

There are **6 features** yielding **2⁶ − 1 = 63** non-empty combinations. Each row shows which features are active, what gets installed, whether an app name is prompted, and key generated files.

**Legend:** S = Standard Django · G = Git · T = Tailwind · R = REST Framework · H = HTMX · A = AllAuth

### 1-Feature Combinations (6)

| # | Features | Pip Packages | npm Packages | App Name Prompted | Key Generated Files |
|---|----------|-------------|--------------|-------------------|---------------------|
| 1 | S | — | — | Yes | `templates/`, `static/`, `context_processors.py` |
| 2 | G | — | — | No | `.gitignore` |
| 3 | T | — | `tailwindcss`, `@tailwindcss/cli` | No | `package.json`, `static/src/styles.css` |
| 4 | R | `djangorestframework` | — | Yes | `serializers.py`, `api.py`, `models.py`, `forms.py` |
| 5 | H | `django-htmx`, `django-browser-reload` | — | Yes | — |
| 6 | A | `django-allauth` | — | Yes | Auth templates |

### 2-Feature Combinations (15)

| # | Features | Pip Packages | npm Packages | App Name Prompted | Key Generated Files |
|---|----------|-------------|--------------|-------------------|---------------------|
| 7 | S + G | — | — | Yes | `templates/`, `static/`, `.gitignore` |
| 8 | S + T | — | `tailwindcss`, `@tailwindcss/cli` | Yes | `templates/`, `static/`, `package.json` (Tailwind-enhanced `base.html`) |
| 9 | S + R | `djangorestframework` | — | Yes | `templates/`, `static/`, `serializers.py`, `api.py`, `api-docs.html` |
| 10 | S + H | `django-htmx`, `django-browser-reload` | — | Yes | `templates/`, `static/` |
| 11 | S + A | `django-allauth` | — | Yes | `templates/`, `static/`, auth templates |
| 12 | G + T | — | `tailwindcss`, `@tailwindcss/cli` | No | `.gitignore`, `package.json` |
| 13 | G + R | `djangorestframework` | — | Yes | `.gitignore`, `serializers.py`, `api.py` |
| 14 | G + H | `django-htmx`, `django-browser-reload` | — | Yes | `.gitignore` |
| 15 | G + A | `django-allauth` | — | Yes | `.gitignore`, auth templates |
| 16 | T + R | `djangorestframework` | `tailwindcss`, `@tailwindcss/cli` | Yes | `package.json`, `serializers.py`, `api.py` (Tailwind-styled forms) |
| 17 | T + H | `django-htmx`, `django-browser-reload` | `tailwindcss`, `@tailwindcss/cli` | Yes | `package.json` |
| 18 | T + A | `django-allauth` | `tailwindcss`, `@tailwindcss/cli` | Yes | `package.json`, auth templates |
| 19 | R + H | `djangorestframework`, `django-htmx`, `django-browser-reload` | — | Yes | `serializers.py`, `api.py` |
| 20 | R + A | `djangorestframework`, `django-allauth` | — | Yes | `serializers.py`, `api.py`, auth templates |
| 21 | H + A | `django-htmx`, `django-browser-reload`, `django-allauth` | — | Yes | Auth templates |

### 3-Feature Combinations (20)

| # | Features | Pip Packages | npm Packages | App Name Prompted | Key Generated Files |
|---|----------|-------------|--------------|-------------------|---------------------|
| 22 | S + G + T | — | `tailwindcss`, `@tailwindcss/cli` | Yes | `templates/`, `static/`, `.gitignore`, `package.json` |
| 23 | S + G + R | `djangorestframework` | — | Yes | `templates/`, `static/`, `.gitignore`, `serializers.py`, `api.py` |
| 24 | S + G + H | `django-htmx`, `django-browser-reload` | — | Yes | `templates/`, `static/`, `.gitignore` |
| 25 | S + G + A | `django-allauth` | — | Yes | `templates/`, `static/`, `.gitignore`, auth templates |
| 26 | S + T + R | `djangorestframework` | `tailwindcss`, `@tailwindcss/cli` | Yes | `templates/`, `static/`, `package.json`, `serializers.py`, `api.py` |
| 27 | S + T + H | `django-htmx`, `django-browser-reload` | `tailwindcss`, `@tailwindcss/cli` | Yes | `templates/`, `static/`, `package.json` |
| 28 | S + T + A | `django-allauth` | `tailwindcss`, `@tailwindcss/cli` | Yes | `templates/`, `static/`, `package.json`, auth templates |
| 29 | S + R + H | `djangorestframework`, `django-htmx`, `django-browser-reload` | — | Yes | `templates/`, `static/`, `serializers.py`, `api.py` |
| 30 | S + R + A | `djangorestframework`, `django-allauth` | — | Yes | `templates/`, `static/`, `serializers.py`, `api.py`, auth templates |
| 31 | S + H + A | `django-htmx`, `django-browser-reload`, `django-allauth` | — | Yes | `templates/`, `static/`, auth templates |
| 32 | G + T + R | `djangorestframework` | `tailwindcss`, `@tailwindcss/cli` | Yes | `.gitignore`, `package.json`, `serializers.py`, `api.py` |
| 33 | G + T + H | `django-htmx`, `django-browser-reload` | `tailwindcss`, `@tailwindcss/cli` | Yes | `.gitignore`, `package.json` |
| 34 | G + T + A | `django-allauth` | `tailwindcss`, `@tailwindcss/cli` | Yes | `.gitignore`, `package.json`, auth templates |
| 35 | G + R + H | `djangorestframework`, `django-htmx`, `django-browser-reload` | — | Yes | `.gitignore`, `serializers.py`, `api.py` |
| 36 | G + R + A | `djangorestframework`, `django-allauth` | — | Yes | `.gitignore`, `serializers.py`, `api.py`, auth templates |
| 37 | G + H + A | `django-htmx`, `django-browser-reload`, `django-allauth` | — | Yes | `.gitignore`, auth templates |
| 38 | T + R + H | `djangorestframework`, `django-htmx`, `django-browser-reload` | `tailwindcss`, `@tailwindcss/cli` | Yes | `package.json`, `serializers.py`, `api.py` |
| 39 | T + R + A | `djangorestframework`, `django-allauth` | `tailwindcss`, `@tailwindcss/cli` | Yes | `package.json`, `serializers.py`, `api.py`, auth templates |
| 40 | T + H + A | `django-htmx`, `django-browser-reload`, `django-allauth` | `tailwindcss`, `@tailwindcss/cli` | Yes | `package.json`, auth templates |
| 41 | R + H + A | `djangorestframework`, `django-htmx`, `django-browser-reload`, `django-allauth` | — | Yes | `serializers.py`, `api.py`, auth templates |

### 4-Feature Combinations (15)

| # | Features | Pip Packages | npm Packages | App Name Prompted | Key Generated Files |
|---|----------|-------------|--------------|-------------------|---------------------|
| 42 | S + G + T + R | `djangorestframework` | `tailwindcss`, `@tailwindcss/cli` | Yes | `templates/`, `static/`, `.gitignore`, `package.json`, `serializers.py`, `api.py` |
| 43 | S + G + T + H | `django-htmx`, `django-browser-reload` | `tailwindcss`, `@tailwindcss/cli` | Yes | `templates/`, `static/`, `.gitignore`, `package.json` |
| 44 | S + G + T + A | `django-allauth` | `tailwindcss`, `@tailwindcss/cli` | Yes | `templates/`, `static/`, `.gitignore`, `package.json`, auth templates |
| 45 | S + G + R + H | `djangorestframework`, `django-htmx`, `django-browser-reload` | — | Yes | `templates/`, `static/`, `.gitignore`, `serializers.py`, `api.py` |
| 46 | S + G + R + A | `djangorestframework`, `django-allauth` | — | Yes | `templates/`, `static/`, `.gitignore`, `serializers.py`, `api.py`, auth templates |
| 47 | S + G + H + A | `django-htmx`, `django-browser-reload`, `django-allauth` | — | Yes | `templates/`, `static/`, `.gitignore`, auth templates |
| 48 | S + T + R + H | `djangorestframework`, `django-htmx`, `django-browser-reload` | `tailwindcss`, `@tailwindcss/cli` | Yes | `templates/`, `static/`, `package.json`, `serializers.py`, `api.py` |
| 49 | S + T + R + A | `djangorestframework`, `django-allauth` | `tailwindcss`, `@tailwindcss/cli` | Yes | `templates/`, `static/`, `package.json`, `serializers.py`, `api.py`, auth templates |
| 50 | S + T + H + A | `django-htmx`, `django-browser-reload`, `django-allauth` | `tailwindcss`, `@tailwindcss/cli` | Yes | `templates/`, `static/`, `package.json`, auth templates |
| 51 | S + R + H + A | `djangorestframework`, `django-htmx`, `django-browser-reload`, `django-allauth` | — | Yes | `templates/`, `static/`, `serializers.py`, `api.py`, auth templates |
| 52 | G + T + R + H | `djangorestframework`, `django-htmx`, `django-browser-reload` | `tailwindcss`, `@tailwindcss/cli` | Yes | `.gitignore`, `package.json`, `serializers.py`, `api.py` |
| 53 | G + T + R + A | `djangorestframework`, `django-allauth` | `tailwindcss`, `@tailwindcss/cli` | Yes | `.gitignore`, `package.json`, `serializers.py`, `api.py`, auth templates |
| 54 | G + T + H + A | `django-htmx`, `django-browser-reload`, `django-allauth` | `tailwindcss`, `@tailwindcss/cli` | Yes | `.gitignore`, `package.json`, auth templates |
| 55 | G + R + H + A | `djangorestframework`, `django-htmx`, `django-browser-reload`, `django-allauth` | — | Yes | `.gitignore`, `serializers.py`, `api.py`, auth templates |
| 56 | T + R + H + A | `djangorestframework`, `django-htmx`, `django-browser-reload`, `django-allauth` | `tailwindcss`, `@tailwindcss/cli` | Yes | `package.json`, `serializers.py`, `api.py`, auth templates |

### 5-Feature Combinations (6)

| # | Features | Pip Packages | npm Packages | App Name Prompted | Key Generated Files |
|---|----------|-------------|--------------|-------------------|---------------------|
| 57 | S + G + T + R + H | `djangorestframework`, `django-htmx`, `django-browser-reload` | `tailwindcss`, `@tailwindcss/cli` | Yes | `templates/`, `static/`, `.gitignore`, `package.json`, `serializers.py`, `api.py` |
| 58 | S + G + T + R + A | `djangorestframework`, `django-allauth` | `tailwindcss`, `@tailwindcss/cli` | Yes | `templates/`, `static/`, `.gitignore`, `package.json`, `serializers.py`, `api.py`, auth templates |
| 59 | S + G + T + H + A | `django-htmx`, `django-browser-reload`, `django-allauth` | `tailwindcss`, `@tailwindcss/cli` | Yes | `templates/`, `static/`, `.gitignore`, `package.json`, auth templates |
| 60 | S + G + R + H + A | `djangorestframework`, `django-htmx`, `django-browser-reload`, `django-allauth` | — | Yes | `templates/`, `static/`, `.gitignore`, `serializers.py`, `api.py`, auth templates |
| 61 | S + T + R + H + A | `djangorestframework`, `django-htmx`, `django-browser-reload`, `django-allauth` | `tailwindcss`, `@tailwindcss/cli` | Yes | `templates/`, `static/`, `package.json`, `serializers.py`, `api.py`, auth templates |
| 62 | G + T + R + H + A | `djangorestframework`, `django-htmx`, `django-browser-reload`, `django-allauth` | `tailwindcss`, `@tailwindcss/cli` | Yes | `.gitignore`, `package.json`, `serializers.py`, `api.py`, auth templates |

### 6-Feature Combination (1)

| # | Features | Pip Packages | npm Packages | App Name Prompted | Key Generated Files |
|---|----------|-------------|--------------|-------------------|---------------------|
| 63 | S + G + T + R + H + A | `djangorestframework`, `django-htmx`, `django-browser-reload`, `django-allauth` | `tailwindcss`, `@tailwindcss/cli` | Yes | `templates/`, `static/`, `.gitignore`, `package.json`, `serializers.py`, `api.py`, auth templates |

### App Name Prompt Summary

| App Name Prompted? | Condition | Combos |
|--------------------|-----------|--------|
| **No** | Only G and/or T selected (no S, R, H, or A) | #2, #3, #12 (3 combos) |
| **Yes** | Any of S, R, H, or A selected | All other 60 combos |

---

## Section 6: Database × Feature Matrix

All **3 database options** work with all **63 feature combinations**. The database choice is independent of feature selection.

| | SQLite | PostgreSQL (Docker) | MySQL (Docker) |
|---|--------|---------------------|----------------|
| **No features selected** (bare Django) | ✓ | ✓ | ✓ |
| **Any of the 63 feature combos** | ✓ | ✓ | ✓ |

### Total Valid Configurations

| Category | Count |
|----------|-------|
| Database options | 3 |
| Feature combinations (non-empty) | 63 |
| Bare configs (database only, no features) | 3 |
| **Feature × Database combos** | 3 × 63 = **189** |
| **Total valid configurations** | 189 + 3 = **192** |

### What the Database Adds

Regardless of which features are selected, the database choice adds:

| Database | Additional Pip Package | Additional Files | Additional `.env` Vars |
|----------|----------------------|-------------------|----------------------|
| SQLite | — | — | — |
| PostgreSQL (Docker) | `psycopg2-binary` | `docker-compose.yml` | `DB_NAME`, `DB_USER`, `DB_PASSWORD`, `DB_HOST`, `DB_PORT` |
| MySQL (Docker) | `mysqlclient` | `docker-compose.yml` | `DB_NAME`, `DB_USER`, `DB_PASSWORD`, `DB_ROOT_PASSWORD`, `DB_HOST`, `DB_PORT` |

---

## Quick Reference: Always Included

Every generated project — regardless of database or feature selection — includes:

| Item | Details |
|------|---------|
| Virtual environment | `.venv/` |
| Django | User-selected version |
| `django-environ` | Environment variable management |
| `python-dotenv` | `.env` file loading |
| `.env` | Environment variables (auto-generated `SECRET_KEY`) |
| `.env.example` | Template for version control |
| `settings.py` | Configured with `environ` integration |
| `manage.py` | Django management script |
| `urls.py` | Project URL configuration |
| VS Code workspace | `.vscode/` (extensions, launch, settings, tasks) |
| `start-dev.sh` | Dev server launcher script |
| `WELCOME.md` | Getting started guide |
| Scaffold formatting | One-time `ruff` + `djlint` cleanup before setup completes |
