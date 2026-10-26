# 9. The spec is typed, and the catalog lists what each choice needs from the machine

Status: accepted, 2026-10-27. Supersedes the `Adjustment` paragraph of ADR 2.

## Context

ADR 2 made `config.Parse` the one place the rules run, but the `Spec` it returned only wrapped the raw `Choices`. Renderers still read strings (`cfg.Name`, `cfg.App`, `cfg.Django`) and asked `cfg.Has(...)`, which recomputed the add-on requirements on every call. Nothing in the types said that a name had been checked or that pre-commit came with git.

The machine checks (git on PATH, Docker ready, Node 20 with npm or pnpm) were written out in preflight. The tool versions the generator pins (Tailwind, the pre-commit hook revisions, djlint for the tests, the Docker engine API) were Go constants, and `make outdated` found them by regex in files that had since been renamed.

## Decision

- `Spec` holds typed fields: `ProjectName`, `PythonSeries`, `DjangoRelease`, `Database`, `Stack`, `Option[AppName]` and a `FeatureSet`. `NewProjectName`, `NewAppName` and `NewDjangoRelease` refuse bad input, and their types have no exported fields, so a value outside the package only comes from a constructor. `Parse` also refuses an unknown stack or database.
- `FeatureSet` is closed under the add-on requirements when it is built. `Spec.Has` is a lookup.
- `Parse` returns the `Adjustment`s it made ("Added git, which pre-commit needs"). The setup summary prints them, and the manifest's `resolved.addedAddOns` comes from them.
- The engine's `project` carries the `Spec`, not the `Choices`. `Choices` is the form the TUI and the answers file fill in before `Parse`.
- `catalog.toml` has a `[tools]` table for the pinned versions and `[[host]]` rows that say what a choice needs from the machine: `addon:git` needs `git`, `database:docker` needs `docker`, and `stack:tailwind` needs `npm|pnpm` and `node>=20`. Preflight walks the rows that apply to the spec. `make outdated` reads the pins from the catalog.
- When the setup starts, it probes the needs in the table (all but Docker, which has its own step). Stacks and add-ons the machine cannot build are shown faint with the reason, and choosing one is refused with the same reason. The add-on list's description is written from the requires and conflicts rows.

## Consequences

- Tests build specs through `Parse`. Test fixtures that described impossible projects (a templates stack without an app) now get an app, which added one line to ten settings goldens.
- A new host need is a catalog row plus, for a new kind of tool, one case in preflight.
- Generated projects are unchanged: the Tier 1 fingerprint and the six real projects match.
