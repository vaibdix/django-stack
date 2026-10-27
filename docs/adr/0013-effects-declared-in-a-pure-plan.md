# 13. Effects are declared in a pure plan

Status: accepted, 2026-10-27. Supersedes the step list and the compensation paragraph of ADR 5.

## Context

ADR 5 made the steps a graph, but each step's needs were still implied: a main-line step needed the step listed before it, and only side branches named their needs. Some parts of an effect were written inside its run function, not declared beside it:

- its timeout (`context.WithTimeout` at the top of each function);
- its undo (`onRollback` inside the database start).

Nothing said which files an effect writes. Git ran on the main line, so creating the repository held up the templates and the apps behind it.

## Decision

- **`internal/plan`** is a pure package. depguard holds it to the same rule as `render`. It lists every effect as data:

  ```go
  type Effect struct {
      ID         string
      Label      string
      Needs      []string
      Produces   []string
      Timeout    time.Duration
      Compensate bool
      When       func(config.Spec) bool
  }
  ```

  Every need is written out. The main line is now an explicit chain in the data, not a rule in the executor.
- **Git is a side branch.** `initialize git` needs only `.gitignore`. `setup pre-commit` needs Git and the dependency manifest. Nothing that writes project files waits for Git, the database or Tailwind, and a test checks that.
- **`Produces`** names what an effect leaves behind:
  - `files:<group>` for each rendered file group;
  - a plain name for anything else (`venv`, `database`, `lock`, `git hooks`).

  `plan.Check` refuses a spec where two applicable effects produce the same thing. `plan.Unwritten` lists any render file group that no effect declares. The executor also enforces the declaration: `emitGroup` refuses a group that the running effect does not declare.
- **`Timeout`** is set on every effect: ten minutes for commands and one minute for file writes. `graph.Run` applies it, so the run functions no longer set their own. Inner timeouts on single commands, such as the lock, the freeze and the hash lookup, stay where they are.
- **`Compensate`** says that an effect leaves state outside the project folder. Only the database start declares it. The executor's table pairs `startDatabaseContainer` with `stopDatabaseContainer` and records the undo only after the start succeeds, which keeps ADR 5's reason for registering it late. The executor refuses to start if the plan and its table disagree about a compensation, or if either lists an effect the other does not.
- **Concurrency limit:** `graph.Hooks.Limit` caps how many effects run at once. The plan sets `Concurrency = 4`, which is enough for the widest point of the graph: the install, Tailwind, the two env files and the compose file. The default is the CPU count.
- The executor (`internal/scaffold`) keeps a map from effect ID to its run and compensate functions. `plan.Count` replaces `scaffold.StepCount` for the TUI's progress bar.

## Consequences

- The order, the parallelism, the timeouts and the undo are visible in one table that can be read and tested without running anything.
- Effects that write project files no longer wait for `git init`.
- Generated projects are unchanged: the Tier 1 fingerprint and the six real projects match.
