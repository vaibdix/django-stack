# 1. Functional core, imperative shell

Status: accepted, 2026-10-26

## Context

`internal/scaffold.Env` holds 163 methods and 17 fields. 117 of the methods are pure: they read the user's choices and return file content. The other 46 run commands, write files, keep the undo stack or hold the mutex. Because all of them share one type:

- a test of a pure renderer must build a whole `Env` (146 `&Env{…}` literals in 48 test files);
- the package cannot be split, so formatting rules, concurrency and rollback live side by side;
- nothing stops a renderer from running a command.

## Decision

Separate what the project contains from how it gets onto disk.

- A pure core turns validated choices into a blueprint: every file, its mode and its content. It does no I/O.
- A thin shell runs the effects (create the folder, install packages, start the database, run migrations, verify) and writes the blueprint.
- Rules (Django series, add-on requirements and conflicts, database images) are data in one catalog, read by the TUI, `-config` and the core alike.
- Add-ons contribute data to the blueprint and declare their effects separately.

The work happens in three tranches on the `v2-architecture` branch. Each tranche lands on `main` by itself:

- A: rules as data and a validated spec; `Env` split into a pure `render` package and an executor; add-ons that return data.
- B: a typed settings and URL model; an effect graph in place of the ordered plan; an event stream with `-json` output.
- C: content checked by its own tools; stage, verify, then publish.

## Consequences

- Generated projects must stay byte-identical through every step. Two checks prove it:
  - `go run ./tools/matrix render-all -check=false -fingerprint <file>` writes one hash per Tier 1 configuration (532,224 at the start), and the file must not change;
  - a small matrix run regenerates six fixed projects with real installs, and `matrix diff` against the run taken before the refactor must show no changes.
- Pure code is tested by calling functions, without fakes or temporary folders.
- A depguard rule keeps I/O packages out of the core.
- The test suite churns while `Env` is split; the test constructor lands first, on its own.
