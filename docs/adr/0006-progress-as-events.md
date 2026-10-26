# 6. Progress is a stream of start and finish events

Status: accepted, 2026-10-27

## Context

`report.Reporter` had `Step(message)` for notes and `Progress(status)` for "a new step started". The CLI printed a step's duration when the next `Progress` arrived, and the TUI advanced its bar on each `Progress`. Both assume one step at a time. Since ADR 5, the database start runs beside other steps, so "time until the next step started" no longer measures anything.

## Decision

- `report.Reporter` is `Step(message)`, `Started(step, label)` and `Finished(step, label, took, err)`. The graph measures `took` around each step's `Run` with the engine's clock.
- `report.Steps` is just `Step(message)`. `pyenv` and `dbdocker` take it, since they only post notes. `pyenv.InstallAll` posted a second `Progress` in the middle of a step; it now posts a note.
- The CLI prints `==> label` when a step starts and `<== label done in 1.50s` (or `failed in`) when it finishes, so overlapping steps each get their own time.
- `-json` (only with `-config`) writes one JSON object per line to stderr: `{"event":"started"|"note"|"finished","step":…,"text":…,"seconds":…,"error":…}`. It is the interface for scripts and CI.
- The TUI shows a step's label when it starts and advances the bar when it finishes. The total is the number of steps that run.

An interface with three methods replaces the plan's channel of `Event` values: every consumer needs the same three cases, and a method call needs no goroutine or buffer.

## Consequences

- Step timings are true durations, including for steps that overlap.
- Scripts can follow a run without parsing the human output.
- The CLI's line format changed. Nothing in the repository parses it, and the end-to-end test only looks for the verification line.
