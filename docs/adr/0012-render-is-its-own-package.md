# 12. Rendering is its own package

Status: accepted, 2026-10-27. Supersedes the last decision of ADR 3, which kept `project` inside `internal/scaffold`.

## Context

ADR 3 made rendering pure inside one package: `project` carried the renderers, and `Env` embedded it. It held back the move to a separate package for one reason: each step wrote its own slice of files, so the executor called about 110 renderer names.

Two changes removed that reason:

- ADR 11 moved the settings and URL order into declared data.
- The steps now write named file groups: `project`, `settings`, `env`, `urls`, `templates`, `app`, `api` and so on. `blueprint()` builds from the same groups, and it fails if two groups write the same path. A step calls `emitGroup(group)`; it no longer calls the renderers one by one.

## Decision

- `internal/render` holds `Project` (the spec, the resolved values, the toolchain and the Node manager), the features, the declared ordering model, the file groups, `Build`, and every pure helper.
- `internal/scaffold` keeps the executor:
  - `Env`, which embeds `render.Project`;
  - the steps and the run graph;
  - preflight and the host probe;
  - the dependency resolve, lock and hashes;
  - the manifest writer, verification and rollback.
- The split was mechanical. A throwaway tool placed each declaration by what it references:
  - anything reached from `Project`, the features or `Build`;
  - anything whose references never reach `Env`, the ports, `os`, `net`, `context`, `sync` or the runner.

  The second rule matters as much as the first: it moved pure helpers that only the executor used, such as the pre-commit config renderer and the package-pin helpers. A method always goes with its receiver's type.
- Tests moved with the code they test. Tests of rendered content build a `project` and write through a plain folder helper. Only executor tests build an `Env`.
- All goldens and the settings fixtures moved to `internal/render/testdata`, so `GOLDEN_PKGS` names `./internal/render`.
- **Enforcement:**
  - The depguard rule `render-is-pure` keeps these out of `internal/render`: `os`, `os/exec`, `net`, `crypto/rand`, `command`, `port`, `adapter`, `releases`, `report` and `scaffold`.
  - `TestNothingInRenderTouchesTheMachine` now follows every function in the package, not only those reached from `project`.
  - The terminal and `exec.Command` guards walk both packages.

## Consequences

- `render` is about 47 files with 215 tests and no fakes. `scaffold` is about 30 files with 77 tests.
- The executor uses 68 exported `render` names. Most of them are file names, package pins and the file-group constants; the rest are the few renderers that need installed versions, such as the pyproject and requirements headers.
- `tools/matrix` and the TUI call `render.Build`, `render.Tools` and `render.NodeManager` directly.
- **Not split further:**
  - New.md also sketches separate `feature` and `model` packages. Features call shared `Project` helpers, so a separate `feature` package would need those helpers exported or would form an import cycle, which is the reason new.md gives for not having one package per feature.
  - The ordering model is about 140 lines that only features use, so it stays in `render`.
- Generated projects are unchanged: the Tier 1 fingerprint and the six real projects match.
