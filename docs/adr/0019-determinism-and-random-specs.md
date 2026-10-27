# 19. Output is proved deterministic, and random specs are rendered on every test run

Status: accepted, 2026-10-27

## Context

new.md lists two tests that the repo did not have:

- **Output determinism:** run `Generate(spec, fixedClock, fixedRand, fixedIndex)` twice and assert byte-identical trees.
- **Property-based rendering:** take random valid specs and check that the rendered files parse, their imports resolve and no two features claim the same slot.

The goldens fix eleven configurations. Tier 1 (`make content-check`) renders every structural configuration, but always with the name `tier-one` and one app name, and it takes minutes. Neither runs `Generate`, so neither shows that the executor adds nothing that varies from run to run.

## Decision

- **`TestTheSameSpecClockAndRandGenerateTheSameProject`** (`internal/scaffold`) runs the whole of `Generate` twice. The two runs use:
  - the same spec;
  - the review clock;
  - a ChaCha8 `Rand` with one seed;
  - a fake runner that answers the interpreter probe and reports what was installed when the freeze runs.

  It then compares the two published trees file by file, with nothing redacted. The manifest's `createdAt` and `indexDate` come from the clock, the secret from `Rand`, and the staging folder's name never reaches a file.
- **`TestTheRandPortIsTheOnlySourceOfTheSecret`** runs the same spec with two seeds. Only `.env` may differ, which shows that the first test passes because the ports are fixed and not because the randomness is ignored.
- **`TestEveryRandomValidSpecRendersWellFormedFiles`** (`internal/render`) takes 300 random valid specs from a seeded generator. Each spec has:
  - a random project name and app name;
  - a Django series and Python minor that the catalog supports;
  - a database, a stack, add-ons, a toolchain and a Node manager.

  For each spec, it builds the blueprint twice and asserts the following:
  - the two builds are identical;
  - every file ends with a newline and has no template residue (`<no value>`, `%!(`, `[[.`);
  - JSON, YAML and TOML files parse;
  - every first-party Python import, absolute or relative, names a module the project contains.

  `Build` itself fails if two features claim the same settings slot (ADR 11), so a collision fails the test too. A second test checks that the seed reaches every stack, database and add-on. A third checks that the checker finds each kind of problem it looks for.
- All three run in `make test` in under two seconds. They need no network, Python or Docker.

## Consequences

- A change that adds map-order or wall-clock output fails in the fast tests, not as noise in a sweep diff.
- Random names cover what Tier 1 does not: names that need a module name different from the project name, and app names up to 31 characters in every file, not only in the templates the live property tests from ADR 7 check.
- The seed is fixed, so a failure reproduces. Changing the seed explores new specs.
