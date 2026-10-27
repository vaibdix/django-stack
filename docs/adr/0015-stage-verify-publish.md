# 15. uv projects are built beside the target and moved in once checked

Status: accepted, 2026-10-27. Supersedes ADR 8.

## Context

ADR 8 kept generation in place for three reasons:

- `pre-commit install` writes the virtual environment's absolute interpreter path into `.git/hooks/pre-commit`.
- pip environments cannot be moved.
- The rollback already removes a failed run's folder.

Two of those reasons have answers:

- ADR 13 made the run a declared plan, so one effect can run after the move.
- A uv environment made with `--relocatable` starts its console scripts relative to their own location.

The third reason still holds, but a project that appears at its final path only once it has passed its checks is easier to trust. It also never shows a half-built folder to an editor or file watcher that is already open there.

## Decision

- **Where a uv run builds:** in a hidden folder beside the target, `.<name>.django-stack-<8 hex>`. The suffix comes from the `Rand` port. The virtual environment is created with `uv venv --relocatable`.
- **What stays in place:** a pip run is built in place, as before, because `python -m venv` writes absolute paths. If uv fails to create the environment and the run falls back to pip, the executor does three things:
  1. It removes the half-made environment.
  2. It moves the staged folder into place at once.
  3. It creates the pip environment there.

  From then on the run is in place.
- **Publishing:**
  - The plan marks effects as `Published`. The executor runs the unpublished effects as one graph, publishes, then runs the published ones. Only `install git hooks` is published; `setup pre-commit` now only writes `.pre-commit-config.yaml`.
  - Publishing first removes every `__pycache__` folder outside `node_modules` and `.git`, because compiled files record the staging path for tracebacks. Then it renames the staging folder onto the target. If the user had made an empty target folder, that folder is checked again and replaced.
  - `FS.Rename` was added to the port, with a contract test for the OS adapter and the in-memory fake.
- **Failures:**
  - A run that fails before publishing removes the staging folder and stops the database. The target never appears.
  - A run whose checks fail (exit 2) is still published, and the hooks are installed.
  - `DJANGO_STACK_KEEP_FAILED=1` publishes the partial project to the target, so the message names a path the user knows.
- **Docker:** Docker is unaffected. The compose file has a fixed `name:` and named volumes, so `docker compose down -v` from the published folder stops what was started in the staging folder.

## Consequences

- A live run (Postgres in Docker, Git, pre-commit, Ruff and mypy) gave these results:
  - No file in the published project, including `.venv`, names the staging folder.
  - `manage.py check` and `django-admin` run from the new path.
  - The hook's `INSTALL_PYTHON` points at the published environment, and `pre-commit run --all-files` passes.
  - `docker compose down -v` there removed the container and its volume.
- A run killed with SIGKILL before publishing leaves a hidden folder beside the target, not a visible half-project at it.
- The progress bar counts one more effect for pre-commit projects.
- The first import after publishing recompiles bytecode, which takes about a second once.
- Generated files are unchanged: the Tier 1 fingerprint and the six real projects match.
