# 18. A committed reference project is checked by its own toolchain

Status: accepted, 2026-10-27. Supersedes the directory-golden and index-name parts of ADR 7.

## Context

ADR 7 checks every unique rendered file with `py_compile`, ruff and djlint. Those checks run file by file, with ruff's error rules only, so they cannot run:

- mypy, which needs the packages and the Django settings;
- the project's own tests;
- Django's migration autodetector.

new.md asks for a reference project, checked into the repo and checked by the real tools. It also asks that Go stop reimplementing what another tool computes. The engine computed Django's md5 index name for the book model's index (`djangoIndexName`).

## Decision

- **`reference/`:** a committed project rendered with every feature on that can run on SQLite:
  - the Tailwind + HTMX stack;
  - every add-on except django-ninja, which conflicts with Django REST framework;
  - uv and pnpm, at a fixed clock and secret.

  It holds the blueprint and `pyproject.toml`. `.env` is left out because it is the secret file, and the check writes it from `.env.example`.
- **The reference cannot drift:** `TestGoldenReferenceProject` renders the project and compares it with `reference/`, file by file, executable bits included. It reports files that are missing or extra. `golden.AssertTree` does the comparison. `make goldens-check` runs it, and `make reference` (or `make goldens`) rewrites the folder.
- **`make content-check`** first runs `render-all` as before. Then it runs `go run ./tools/matrix reference` on a temporary copy of `reference/`, which does these steps:
  1. `uv sync`;
  2. `ruff format --check` and `ruff check`, with the project's own configuration;
  3. `mypy` (strict, with `disallow_any_explicit`);
  4. `djlint --lint --check`;
  5. `manage.py check`;
  6. `makemigrations --check --dry-run`;
  7. `manage.py test`.

  The project's tests use Django's test runner, so the check runs that runner and not pytest.
- **Index name:** the book model names its index, `book_created_at_idx`. The rendered migration repeats that literal, and `djangoIndexName` is removed. Django's autodetector writes the name as given, so Go no longer computes it. `makemigrations --check` on the reference fails if the two ever disagree.
- **What stays in Go:**
  - the migration import order, which depends on the app name;
  - djlint's attribute wrapping (`tag`).

  Neither can be fixed by a literal, and running Django or djlint during generation would make every run slower. Their live property tests from ADR 7 stay.

## Consequences

- The reference check takes about 20 seconds on a warm uv cache. It covers 45 source files under mypy and runs 34 tests. Renaming the index in `models.py` alone makes it fail with a `0002_rename_…` migration.
- Every generated project's book index changes name, from `<app>_boo_created_<hash>_idx` to `book_created_at_idx`. No other line changes. Index names are global within a PostgreSQL schema, and a fresh project has one book model, so the fixed name cannot collide.
- The repo carries about 80 rendered files under `reference/`. A diff to them in review shows the effect of a template change on a real project.
