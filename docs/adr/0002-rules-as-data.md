# 2. Rules as data, and a parsed spec

Status: accepted, 2026-10-26

## Context

The rules that decide what a valid project is lived in Go tables and in checks spread over three callers:

- the Django series and the Python versions each supports, add-on labels, requirements and conflicts, stacks, database images and drivers, and the module names a project must not shadow were Go literals in `internal/config`;
- the name, app and Django-on-Python checks ran in `Answers.Choices`, in the `-config` path after it found an interpreter, and again in the engine's preflight, each with its own wording;
- the engine accepted any `config.Choices`, so nothing proved the checks had run.

## Decision

- `internal/catalog/catalog.toml` holds the rule tables. `catalog.Load` parses the embedded file once with unknown keys refused, checks that names are unique, that every requirement and conflict names a real add-on, that conflicts are listed on both sides and that support dates are a year and month. A malformed catalog panics at start-up, and a test parses it on every run.
- `internal/config` keeps its API (`AddOns`, `Stack.Spec`, `Database.Spec`, `DjangoRunsOn` and the rest), now built from the catalog.
- `config.Parse(choices, interpreter, today)` runs every rule in one place: Django chosen, project name, app name and whether one is needed, add-on conflicts, and Django on the chosen Python. A broken rule is a `*config.RuleError` that names the field.
- `Parse` returns a `config.Spec`. `scaffold.New`, `scaffold.Preflight` and `scaffold.Build` take a `Spec`, so the engine can only run on choices that passed the rules. Preflight keeps only the checks that need the machine: Git, Docker, Node and the Python toolchain.
- The answers file, the `-config` path and the TUI all call `Parse`. `config.AnswerError` words a rule error with the answers-file key (`invalid "app": …`).

The plan proposed an `Adjustment` type for the add-ons that requirements switch on. `Choices.AddedAddOns` already gives that list to the manifest and the TUI, so it stays.

## Consequences

- A new series, add-on, stack or database starts as a catalog entry, and the TUI, `-config` and the matrix tools see it at once.
- The engine's tests build a `Spec` with a helper that runs `Parse`, so a test fixture that breaks a rule fails at once instead of exercising an impossible project.
- A fuzz test checks that every spec `Parse` accepts survives a round trip through its answers.
