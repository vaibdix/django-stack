# 7. Generated content is checked by the tools that will run on it

Status: accepted, 2026-10-27. The directory goldens and the index name are superseded by ADR 18.

## Context

About 130 lines of generated Python, TOML and JavaScript were Go string literals: the single-file production block, the env and database block, the context processor, `main.js`, the Swagger CSP wrapper, and the mypy, ruff, Sentry and requirements fragments. The engine also reimplements two rules from other tools, because running those tools during generation would be slow or impossible:

- djlint's attribute wrapping (the `tag` template function);
- Django's index names (`djangoIndexName` for the rendered migrations).

Both were checked only for the app name `catalog`.

## Decision

- Every multi-line piece of generated content is a template under `internal/assets/templates/`. The Go functions that wrote it keep their names and render the template, and the "every template is rendered" guard covers the new files. Output stayed byte for byte the same (goldens, the Tier 1 fingerprint and six real projects).
- `make content-check` renders every Tier 1 configuration into a temporary folder and runs `py_compile`, ruff (format, imports and errors, per Python minor), `djlint --check` and `docker compose config` on each unique file, then removes the folder. It reuses `tools/matrix render-all` and takes minutes, so it is not part of `make ci`.
- The plan's directory goldens were not added: `render-all` already writes every generated file to disk for the tools, across all configurations, not only the golden ones.
- Two live property tests render random app names (seeded, so a failure reproduces):
  - `TestTemplatesNeedNoDjlintReformatForAnyAppName` runs `djlint --check` on every template for 10 names of 1 to 60 characters;
  - `TestRenderedMigrationsMatchDjangoForAnyAppName` runs `makemigrations --check` for 4 names.

## Consequences

- The djlint property test found a real bug on its first run. The book pages hard-coded wrapping that fit `catalog` only:
  - djlint 1.47 spreads attributes at 70 characters and over, not over 70;
  - it moves an `{% include %}` to its own line once the line's text, without indentation, reaches 120 characters.

  A project with any other app name had its templates rewritten by its own pre-commit hook on the first commit. All 16 such tags now go through `tag`, a `fits` helper encodes the line rule, and `catalog` output is unchanged.
- A change to djlint's or Django's rules shows up as a failing live test, not as a failed first commit in a user's project.
