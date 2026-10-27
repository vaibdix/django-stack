# 16. The manifest has a schema version, and a run resolves at a recorded index date

Status: accepted, 2026-10-27

## Context

`django-stack.json` records the answers that made a project, so that `-config` can generate it again. It had no version field. When its name changed from `.jsonc` to `.json`, nothing in the file told a reader which shape it had. Replaying a manifest months later also resolved every package against the index of that day, so the lock and the hashes differed from the original even with the same answers.

## Decision

- **`schemaVersion`:** the manifest's first field is `schemaVersion`, now `1`. When `config.DecodeAnswers` reads a manifest:
  - it treats a missing field as schema 0;
  - it applies one migration per version up to `config.ManifestSchema`;
  - it refuses a schema newer than it knows, and the message says to update django-stack.

  Schema 0 to 1 only adds the field, because the answers did not change shape. The migration exists so the next change has a place to go, and a test runs every older schema forward.
- **`indexDate`:** the answers have an optional `indexDate`, either a date (`2026-10-27`) or an RFC 3339 time. When none is given, a run resolves at the moment it started, to the second. Either way, the manifest records the date it used, so a replay resolves against the same index.
- **How each toolchain uses the date:**
  - **uv:** the install runs `uv pip install --exclude-newer <date>`. The lock runs twice: first `uv lock --exclude-newer <date>` to pick the versions, then a plain `uv lock`. The second lock keeps every locked version and drops the date from `uv.lock`'s options. If the date stayed there, the project's own `uv lock --check` (its pre-commit hook) would fail, because that check runs without the date.
  - **pip:** the install adds `--uploaded-prior-to <date>` when the environment's pip supports it. The run checks `pip install --help` once to find out. The pip bundled with older interpreters, such as pip 23 on Python 3.10, does not support it. Those runs say that they resolve against today's index.
- **The sweep:** `tools/matrix select` writes the run's `ExcludeNewer` date into every answers file. The flag and the environment variable then agree.

## Consequences

- Each generated manifest gains two lines. The six-project pipeline showed exactly these lines and nothing else: same locks, same requirements and same hashes for uv and pip.
- A project generated from a replayed manifest resolves the same versions, as long as the index still serves them.
