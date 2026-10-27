# 14. The engine sends events on a channel, with command output

Status: accepted, 2026-10-27. Supersedes the interface decision of ADR 6.

## Context

ADR 6 kept a three-method `Reporter` interface instead of a channel. Its reason was that every consumer needed the same three cases. Two things changed that:

- The output of the commands a run starts (uv, pip, npm, Django) was only seen when a command failed. Showing it as it arrives is a fourth case, and only some consumers want it.
- The TUI turned each `Reporter` call into a Bubble Tea message on its own channel anyway, with a `done` guard. That is a channel consumer written as an interface.

## Decision

- `internal/event` replaces `internal/report`. It has four event types: `Started{Effect, Label, At}`, `Output{Effect, Line}`, `Finished{Effect, Label, Took, Err}` and `Note{Text}`. Their marker method is unexported, so the set is closed. `event.Notes` (`Step(message)`) stays for `pyenv`, `dbdocker` and the Docker port, which only post notes.
- `scaffold.Options.Events` is a `chan<- event.Event`. A nil channel discards everything. The caller owns the channel: it drains it while `Generate` runs and closes it afterwards.
- **Command output:**
  - `port.LineRunner` adds `RunLines(ctx, dir, line, name, args…)`. `command.OS` implements it with one writer for stdout and stderr, so it still returns the whole combined output for errors.
  - The executor's runner sends each line as an `Output` event. It does so only when the runner can stream, a consumer is listening and the command runs inside an effect, so every line is tagged with its effect.
- **`internal/ui/text`** holds the two plain consumers:
  - `Lines` prints the human format from ADR 6. It adds `    | line` for command output only with the new `-verbose` flag.
  - `JSON` writes one object per event. Output lines are `{"event":"output","step":…,"text":…}`.

  A depguard rule lets the package import nothing but `event` and the standard library.
- **`internal/ui/tui`** (moved from `internal/tui`) forwards engine events to Bubble Tea messages. It skips `Output`, so the screens are unchanged. When the TUI has gone, the forwarder keeps draining, so a cancelled run can never block on a send. A depguard rule keeps `os/exec` and `adapter` out of the TUI.

## Consequences

- `-config … -json` and `-config … -verbose` show what each tool printed, tagged with its effect.
- **Not adopted:** new.md also wants the UI to know nothing Django-specific except the spec and the events. The setup screens still ask `pyenv`, `dbdocker` and `releases` for the interpreters, Docker's state and the Django releases they offer. Moving those probes behind the engine would change every setup screen, for no change in what the user sees.
- Generated projects are unchanged.
