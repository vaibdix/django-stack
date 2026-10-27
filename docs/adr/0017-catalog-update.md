# 17. `make catalog-update` rewrites the catalog from its sources

Status: accepted, 2026-10-27

## Context

ADR 9 moved the pinned tool versions and the Django support table into `internal/catalog/catalog.toml`. `make outdated` reports which pins are behind, but a person still has to look up each version and edit the file by hand. The Django rows (series, Python range, end of support) were also edited by hand from the Django download page.

## Decision

- `make catalog-update` runs `tools/outdated -write`. It rewrites the catalog in place and prints each change:
  - **Tool pins:** Tailwind, the pre-commit hooks, django-upgrade and djlint move to their latest release from npm, GitHub and PyPI. Names in `OUTDATED_HOLDS` are skipped.
  - **Support dates:** each Django series' `supported_through` follows the end-of-life month in the release schedule at `endoflife.date/api/django.json`.
  - **New series:** a series that has been released and is newer than every row is added at the top. It copies the newest row's Python range, and the command prints a reminder to check that range against the release notes.
- Docker images are only reported. Docker Hub gives a digest for `latest`, not a version number to write.
- The edits are textual, so the file keeps its layout and the diff shows only the changed values. The rewrite functions are tested against fixed schedules.

## Consequences

- An update is one command, then a reviewed diff, `make ci` and `make test-live`.
- A new Python range is still a human decision, which the printed reminder points to.
