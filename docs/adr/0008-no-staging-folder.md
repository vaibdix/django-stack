# 8. Projects are generated in place, not staged and renamed

Status: accepted, 2026-10-27

## Context

The plan proposed generating each project in a hidden staging folder next to the target, verifying it there, and renaming it into place. A failed run would then leave nothing at the target. Virtual environments record absolute paths, so the plan limited staging to uv projects with `uv venv --relocatable`, and pip projects would keep generating in place.

## Findings

- `pre-commit install` writes the virtual environment's absolute interpreter path into `.git/hooks/pre-commit` (`INSTALL_PYTHON=/…/.venv/bin/python`). A relocatable environment does not change that, so a renamed project's first commit would run a hook that points into a folder that no longer exists. Installing hooks after the rename would move that step out of the verified run.
- pip environments cannot be moved at all, so the engine would keep two ways to place a project, chosen by the toolchain.
- The rollback already removes a failed run's folder (or empties a folder that existed before) and stops its database container, newest first. The only case staging improves is a process killed before rollback can run, and then it leaves a hidden folder instead of a visible one.

## Decision

Step 8 is not implemented. Projects are generated in place, with the rollback from ADR 5 undoing a failed run.

## Consequences

- One placement path for both toolchains.
- A run killed with SIGKILL still leaves a partial project. Running again with the same name reports that the folder is not empty.
