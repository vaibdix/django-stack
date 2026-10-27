# 3. The project renders purely

Status: accepted, 2026-10-27. ADR 12 supersedes its last decision: rendering now has its own package.

## Context

ADR 1 separates what a project contains from how it gets onto disk. The plan was to move every pure renderer into a new `internal/render` package, with a depguard rule that keeps I/O packages out of it.

Measured after the split into `project` and `Env`, the executor calls 30 renderer methods and about 80 shared helpers: file-name constants, package pins, `NodeManager`, `fileSet`, the pre-commit and manifest helpers. It does so because each step writes its own slice of files (`.env` now, `urls.py` later, the README near the end). A separate package would have to export about 110 names, which is a wide interface that hides very little.

## Decision

- `project` (the choices, the resolved values, the toolchain and the Node manager) carries the 92 methods that render content. `Env` embeds it and adds the runner, the folder, the undo stack, the lock and the recorded state. A renderer cannot reach the runner or the folder, because `project` has no such fields.
- Add-ons are `feature` values with an `enabled` rule and a `contribute(project) contribution` function. They return data and run nothing. The six add-on steps (Tailwind setup and build, Git, AllAuth, the two API frameworks, pre-commit) are listed in `plan()` in run order, so the whole run order is in one place.
- `TestTheProjectRendersWithoutTouchingTheMachine` type-checks the package and follows every call from a `project` method through the package's functions. It fails if any path reaches `os`, `os/exec`, `net`, `net/http`, `crypto/rand`, `math/rand/v2`, `time.Now`, or the `command`, `dbdocker`, `pypi` or `releases` packages. Two pure helpers, `dbdocker.Compose` and `dbdocker.DatabaseName`, are allowed by name. The test prints the call path of each violation.
- The first run of that test found two methods that read the machine (`indexFor` read an environment variable, and `nodeDependencies` read `package.json`). Both moved to `Env`.
- The move to a separate package waits until the executor writes named groups of the blueprint, not individual renderer results (see ADR 4). At that point the interface is `Build` plus a few renderers that need installed versions, and the move becomes cheap.

## Consequences

- The purity rule holds today and is enforced by a test, not by convention.
- `internal/scaffold` stays one package of about 50 files for now.
- Go 1.27 lets a struct literal set the embedded `project`'s fields directly, so the tests' `&Env{cfg: …}` literals did not change. The planned test constructor was not needed.
