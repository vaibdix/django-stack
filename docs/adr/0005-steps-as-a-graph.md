# 5. Steps run as a dependency graph

Status: accepted, 2026-10-27

## Context

`plan()` was an ordered list. Two Tailwind steps carried an `alongside` flag that ran them in parallel with the step before them; everything else ran one at a time. The order a step needed was implied by its position, and a test (`TestLoadBearingOrderingsHold`) listed the positions that mattered. The Docker database start (about 9.5 s) waited for the virtual environment, the install and every file before it, although it needs only `.env` and the compose file.

## Decision

- `internal/graph` runs a list of nodes. A node starts when every node it needs has finished. A need must name a node listed earlier, so the list is always a valid order and a cycle cannot be written. On the first failure the run is cancelled, the running nodes are awaited, and the error names the node that failed first, not a sibling that was cancelled. A skipped node finishes as soon as its own needs have, so a dependency never jumps past a step that is not running.
- `plan()` is a main line plus side branches. A main-line step needs the main-line step before it. A side branch needs only what it names: Tailwind setup, `.env`, `.env.example`, the compose file, the database start (which needs `.env` and the compose file) and the Tailwind build (which needs the setup and the dependency manifest). Migrations need `.env` and the database. The stack manifest needs the Tailwind steps and `.env.example`. The checks come last.
- The secret key and the database credentials are chosen before the graph starts. `project` is copied by value into every renderer, so it must not change while steps run in parallel.
- Creating the virtual environment can switch the toolchain from uv to pip, and the toolchain changes rendered files. Every step therefore needs the virtual environment, and a test checks it.
- `TestLoadBearingOrderingsHold` now checks that each constraint holds in the graph, directly or through other steps, and that the checks need every step.
- Compensation stays where it is: a step that creates something registers its undo with `onRollback` when it has done so, and a failed run undoes them newest first. The plan's declared `Compensate` field would run an undo even for a step that never got that far.

## Consequences

- The database starts while the install, the files, the lock and pre-commit run.
- The progress bar counts every step, including the two Tailwind steps, so the TUI's step totals changed.
- Progress messages from parallel steps interleave. ADR 6 replaces the per-step timing, which assumed one step at a time.
