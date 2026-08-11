# django-stack — Codebase Audit

**Date:** 2026-08-10
**Commit audited:** `642b5b4` (`init`, single commit, branch `main`)
**Scope:** Full repository — 55 Go files, ~8,400 LoC, plus build tooling, docs and repo hygiene.
**Reviewer stance:** Staff-engineer review against Go community and Google Go style conventions.

> **Addendum (2026-08-10, post-verification).** A second full read-through plus the Phase 0/1 remediation
> work found five defects the first pass missed and narrowed one claim. See
> [§10 Addendum](#10-addendum--findings-from-re-verification-and-remediation) at the end of this document.
> Remediation is tracked in `~/.claude/plans/curious-cooking-zebra.md`.

---

## 1. Executive summary

`django-stack` is a Bubble Tea TUI that scaffolds Django projects. The product logic is genuinely good: the feature matrix is broad, the ordering constraints between generators (multi-env last, pre-commit after ruff) are understood and commented, and the existing 52 tests are behaviour-focused with descriptive names — better than most projects at this stage.

The problems are **structural, not logical**. Three stand out:

1. **A data race at the heart of the app.** `CreateProject(m)` runs in a goroutine and mutates the same `*Model` the Bubble Tea event loop reads and writes. Two fields are mutex-guarded; roughly thirty are not. This is invisible today only because no test exercises the concurrent path.
2. **One 7,379-line package doing everything.** `internal/ui` mixes terminal rendering, subprocess orchestration, network calls, and file generation. Nothing can be tested without the TUI, so nothing is — 239 of 263 functions had 0% coverage.
3. **Repo hygiene that blocks the obvious next steps.** `.gitignore` ignores `.github/`, so CI can never be committed. Two multi-megabyte binaries and a SQLite database are tracked. The module path isn't a URL, so `go install` cannot work.

None of these require a rewrite. The fixes are mechanical and independently shippable; §9 sequences them.

### Scorecard

| Dimension | Grade | One-line verdict |
|---|:---:|---|
| Correctness / concurrency | **D** | Unsynchronised shared mutable state between goroutine and UI loop |
| Package structure | **D** | One god package; two near-identically-named packages; two dead ones |
| Testing | **C−** | Good tests where they exist; 5.9% coverage, no CI, no `-race`, no `t.Parallel` |
| Error handling | **C+** | Good `%w` discipline in `internal/`, absent in `cmd/`; 8 dropped errors |
| Tooling / CI | **F** | No CI, no linter config, no `.editorconfig`; CI is actively gitignored |
| Repo hygiene | **D** | 25 MB of tracked build artefacts; committed DB; stray venv on disk |
| Docs | **B+** | README and OPTIONS.md are thorough and current; zero package/API docs |
| Security posture | **C** | No injection reachable from user input, but 21 gosec findings warrant triage |

**Verified baseline (before this pass):** `go build ./...` clean · `go vet ./...` clean · `go test ./...` passing · `gofmt -l` flags 5 files · `staticcheck` 3 issues · `golangci-lint` 105 issues.

---

## 2. Repository inventory

```
django-cli/
├── cmd/cli/                    3 files    entry point, --install for unix/windows
├── internal/
│   ├── errors/       133 LoC   CLIError + 14 sentinels    ← imported by nothing
│   ├── logger/       252 LoC   levelled logger, JSON mode ← imported by nothing
│   ├── template/      35 LoC   //go:embed CSS loader
│   ├── templates/   ~900 LoC   Go string constants for HTML
│   └── ui/         7,379 LoC   40 files, ONE package
├── test/ui/        1,052 LoC   external black-box test package
├── scripts/                    manual_smoke_tui.sh
├── makefile, build-all.sh, colors.yaml
├── README.md (20 KB), OPTIONS.md (15 KB), CHANGELOG.md, CLAUDE.md
└── ── tracked build artefacts ──
    ├── cli                 11 MB   compiled macOS binary
    ├── django-stack.exe    13 MB   compiled Windows binary
    └── db.sqlite3         136 KB   a Django dev database
```

Ten largest source files: `test/ui/ui_test.go` (1052), `api_templates.go` (685), `settings.go` (492), `django_versions.go` (344), `templates/content.go` (343), `python_options.go` (340), `api_generators.go` (314), `env.go` (306), `python.go` (304), `multienv.go` (299).

---

## 3. Critical findings

### C1 — Data race: `CreateProject` mutates `*Model` concurrently with the UI loop

**Evidence:** `internal/ui/update.go:149-151` and `update.go:171-173`

```go
go func() {
    _ = CreateProject(m)
}()
```

`CreateProject` (`creator.go:9`) writes `m.TotalSteps` and `m.CompletedSteps` (lines 45–46) and calls `m.AppendStepMessage` ~30 times. Meanwhile the Bubble Tea loop writes `m.TotalSteps` at `update.go:146` and `update.go:168`, and `View` reads `m.StepMessages` at `view.go:101-107`.

Only `CompletedSteps` (via `progressMu`) and `StepMessages` (via `StepMessagesMu`) are guarded. `TotalSteps`, `ProgressStatus`, `Error`, `Done`, `Step`, `Width`, and every `ProjectConfig` field are read and written across the goroutine boundary with no synchronisation.

`view.go:101-107` is the sharpest edge — it reads `len(m.StepMessages)` and slices it **without** taking `StepMessagesMu`, while the generator goroutine appends. A concurrent append that reallocates the backing array can produce a torn read or an out-of-range slice.

**Why it matters:** Go's memory model gives no guarantees here. Symptoms are non-deterministic: stale progress, a panic in `View` under load, or a lost final error. `go test -race` does not catch it because no test drives the concurrent path — absence of a race report is not evidence of absence.

**Fix — the Bubble Tea way.** The model must only ever be mutated inside `Update`. Move generation off the model entirely:

```go
// Generation reads an immutable config and reports progress over a channel.
func Generate(ctx context.Context, cfg ProjectConfig, progress chan<- ProgressMsg) error

// Update owns all state transitions:
case StepProjectConfig:
    cfg := m.ProjectConfig          // copy, not share
    cmds = append(cmds, func() tea.Msg {
        return startGenerationMsg{cfg: cfg}
    })
```

Have the generator send `tea.Msg` values via `Program.Send` (already available as `m.Program`) and let `Update` apply them. Once the goroutine no longer touches `m`, both mutexes and `test_exports.go`'s concurrency shims can be deleted.

**Interim mitigation** if the refactor is deferred: guard `View`'s `StepMessages` access with `StepMessagesMu` and add a `Snapshot()` accessor. This narrows the window; it does not close it.

---

### C2 — `View` mutates model state

**Evidence:** `internal/ui/view.go:86-92`

```go
m.Progress.Width = contentWidth - 8
if m.Progress.Width > 80 { m.Progress.Width = 80 }
```

`View` must be a pure function of model state. Bubble Tea may call it at any time relative to `Update`; combined with C1 this is a second unsynchronised writer. The identical clamping logic already exists correctly in `Update`'s `tea.WindowSizeMsg` handler (`update.go:36-44`) — this block is a duplicate that should simply be deleted.

---

### C3 — CI can never be committed

**Evidence:** `.gitignore`

```
coverage.txt
dist/
.DS_Store
.claude/
.github/          ← no trailing newline, and it ignores all CI config
```

`.github/` is gitignored, so no workflow file can ever be added. This single line is why there is no CI, and it silently defeats any attempt to add one. `.claude/` being ignored is reasonable; `.github/` is almost certainly a copy-paste error.

**Fix:** remove `.github/` from `.gitignore`, add a trailing newline, and extend it (see §7.3).

---

## 4. High-severity findings

### H1 — 25 MB of build artefacts tracked in git

`git ls-files` shows `cli` (11 MB), `django-stack.exe` (13 MB) and `db.sqlite3` (136 KB) are tracked. Binaries in git bloat every clone forever, produce unreadable diffs, and — for a tool that installs itself into `/usr/local/bin` — invite someone to run a binary nobody can audit against the source.

`db.sqlite3` is a Django development database that has no business in a Go CLI's repository at all.

**Fix:**
```bash
git rm --cached cli django-stack.exe db.sqlite3
```
then add them to `.gitignore`. Since history is a single `init` commit, this is the cheapest it will ever be. Ship release binaries via GitHub Releases / GoReleaser instead.

### H2 — Module path is not a URL

**Evidence:** `go.mod:1` — `module django-stack`, while `README.md` documents `git clone https://github.com/vaibdix/django-stack.git`.

A non-URL module path means `go install github.com/vaibdix/django-stack/cmd/cli@latest` cannot work, and the module can never be a dependency of anything. This is precisely the friction the custom `--install` flag in `cmd/cli/install_unix.go` was written to work around — fixing the module path removes the need for that code path.

**Fix:** `module github.com/vaibdix/django-stack`, update the four internal import prefixes, and document `go install`.

### H3 — Test-only code ships in the production binary

**Evidence:** `internal/ui/test_exports.go` — 194 LoC, 29 exported `…ForTest` functions, `package ui`, **no build tag, not a `_test.go` file**.

This compiles into every shipped binary. It also exports mutable seams (`SetPortListenerForTest`, `loadPythonOptions`, `fetchDjangoReleaseOptions`) as part of the package's public API surface.

The root cause is that `test/ui/ui_test.go` is `package ui_test` in a *separate directory*, so it cannot reach package internals and needs shims.

**Fix (the Go idiom):** move the tests into `internal/ui/` and rename the shim file `export_test.go`. Files named `*_test.go` are compiled only during testing, so the exports vanish from the shipped binary and the `ForTest` suffixes can be dropped. The `test/` directory should hold only genuine end-to-end tests that exercise the built binary.

### H4 — Two dead packages (385 LoC)

`internal/errors` and `internal/logger` are imported by **zero** files:

```
$ grep -rn 'internal/logger' --include='*.go' .   →  (no output)
$ grep -rn 'internal/errors' --include='*.go' .   →  (no output)
```

Meanwhile `internal/ui` calls `fmt.Print*` 16 times and builds errors with bare `fmt.Errorf`. So the project has a well-designed error taxonomy with hint text (14 sentinels, E001–E014) and a levelled logger — and uses neither.

**Decide and act:** either wire `internal/errors` into the generators (the hints are genuinely good UX and `CLIError.Print()` is exactly what the failure screen needs), or delete both packages. Leaving them is the worst option — they read as live infrastructure and will drift.

Note `logger` is additionally the wrong tool for a full-screen TUI: writing to stdout corrupts the alt-screen. If you keep it, scope it to a `--verbose` non-TUI mode or point it at a log file.

### H5 — Zero coverage of the orchestration core

Before this pass: **5.9% total**, 239 of 263 functions at 0%. `CreateProject` (cyclomatic complexity **62**), `Update` (**51**) and `View` (**35**) — the three functions where bugs actually live — had none.

Complexity that high is itself the finding: `CreateProject` is a 260-line straight-line script with 25 `if` blocks and a `defer` that doubles as the error channel. It cannot be unit-tested as written.

**Fix:** extract each feature into a `Step` with a uniform signature, then table-test the plan:

```go
type Step struct {
    Name    string
    Enabled func(ProjectConfig) bool
    Run     func(context.Context, Env) error
}

func Plan(cfg ProjectConfig) []Step   // pure — trivially testable
```

`CalculateTotalSteps` then becomes `len(Plan(cfg))` rather than a hand-maintained parallel list of `StepCount*` constants that can silently drift out of sync with the actual work performed.

---

## 5. Medium-severity findings

### M1 — Dropped errors (8, via errcheck)

| Location | Dropped call | Consequence |
|---|---|---|
| `internal/ui/multienv.go:84` | `os.WriteFile(envPath, …, 0600)` | `DJANGO_SETTINGS_MODULE` silently missing → project won't boot |
| `internal/ui/docker.go:288,291` | `cmd.Run()` (docker stop/rm) | Container leak reported as success |
| `internal/ui/docker.go:62` | `listener.Close()` | FD leak in the port scan loop |
| `internal/ui/django_versions.go:172` | `resp.Body.Close()` | Connection leak |
| `internal/logger/logger.go:110,133,139` | `fmt.Fprintf` | Acceptable; silence explicitly with `_, _ =` |

M1's first row is the one that bites users: multi-env setup reports success while producing a project that cannot start.

### M2 — TOCTOU in port allocation

**Evidence:** `internal/ui/docker.go:55-62`

```go
listener, err := listenTCP("tcp", fmt.Sprintf("127.0.0.1:%d", port))
if err != nil { continue }
_ = listener.Close()
return strconv.Itoa(port), nil   // port is free… for now
```

The port is released before Docker binds it. On a busy machine two rapid runs can be handed the same port. Unavoidable in full generality, but the failure should be handled: check `docker compose up` output for a bind failure and retry with the next port, rather than surfacing a raw Docker error.

### M3 — `NewModel()` performs subprocess and network I/O

`startup.go:57-58` calls `RefreshPythonOptions()` (which shells out to `which`/`where`/`pyenv`) and `RefreshDjangoOptions()` (which HTTP-GETs `pypi.org`). Constructing the model therefore takes a network round trip and cannot be done in a hermetic test — which is why the new `model_test.go` builds `Model` literals directly.

**Fix:** return the model synchronously and fetch options as `tea.Cmd`s dispatched from `Init()`. That is the idiomatic Bubble Tea pattern, it makes the model constructible in tests, and it stops a slow DNS lookup from blocking the splash screen.

### M4 — Uncancellable long-running work

`Ctrl+C` during setup returns `tea.Quit` (`update.go:20-24`) but the generator goroutine keeps running: it holds no `context.Context` from the UI. Each subprocess creates its *own* `context.WithTimeout(context.Background(), …)` (`venv.go:11`, `model.go:28`), so nothing propagates cancellation.

The TUI exits while `pip install` continues in the background and keeps writing to a half-built project directory.

**Fix:** create one `context.Context` in `main`, thread it through `Generate(ctx, …)`, and cancel it on quit. Google's style guide is explicit that `context.Context` belongs as the first parameter of any function that does I/O.

### M5 — `noctx`: three subprocess calls and one HTTP call without context

`docker.go:20,25` (`docker --version`, `docker info`), `migrations.go:15` (`git init`), and `django_versions.go:163` (`http.NewRequest` instead of `NewRequestWithContext`). A hung Docker daemon hangs the TUI with no timeout and no way out.

### M6 — Two packages named `template` and `templates`

`internal/template` (embedded CSS) and `internal/templates` (HTML string constants) differ by one character and are both imported by `internal/ui/utils.go` (lines 9–10). Readers cannot tell them apart at the call site, and `utils.go` exists largely to re-export one through the other.

**Fix:** merge into a single `internal/assets` package, or rename to `internal/css` and `internal/htmltmpl`.

### M7 — Django/HTML generated by string concatenation

`api_templates.go` has 68 raw-string literals; `api_generators.go` 30; `templates/content.go` 27. There are 25 `generate*` functions producing Python and HTML via `fmt.Sprintf`.

Two consequences: no compile-time or test-time validation that the output parses, and `text/template` + `//go:embed` would make these files diffable, syntax-highlightable, and reviewable. `internal/template/loader.go` already demonstrates the `//go:embed` pattern — extend it to `.py.tmpl` and `.html.tmpl` files.

Also note `staticcheck` flags `basicBaseTemplate` and `basicIndexTemplate` in `templates/content.go` as entirely unused, plus `writeFile` in `utils.go`.

### M8 — `ProcessFormData` couples logic to display strings

`update.go:216-239` switches on user-facing labels (`"Standard Django Project"`, `"Pre-commit"`) that are defined 130 lines away in `startup.go:78-87`. Renaming a label in the UI silently disables the feature — no compiler error, no test failure (until now; `model_test.go` covers this mapping).

**Fix:** make the `huh.Option` values a typed constant (`type Feature string`) and switch on that. The label becomes free-text.

---

## 6. Low-severity findings

- **`gofmt` fails on 5 files** — `api.go`, `app.go`, `constants.go`, `multienv.go`, `startup.go`. `startup.go:83` has visibly broken indentation. Fix: `gofmt -w ./cmd ./internal`. This is table stakes; every Go CI gates on it.
- **`ST1005` — capitalized error strings** at `docker.go:73,77,175,179` and `venv.go:101,108`. Go error strings start lowercase because they get wrapped: `failed to X: Docker is not running`. (`Python not found` is fine — proper noun.)
- **`errorlint` — `%v` instead of `%w`** at `cmd/cli/install_unix.go:14,22,26` and 5 further sites. `internal/` uses `%w` correctly and `cmd/` does not, so the wrapping chain breaks exactly at the boundary where the user sees it.
- **`unparam` — 4 dead parameters**: `generateBookForms(appName)`, `compatibilityLabel(version)`, `generateWindowsStartScript(useTailwind)`, `generateUnixStartScript(projectPath)`.
- **`gocritic` — 3 if-else chains** in `settings.go:98,150,372` that should be `switch`; `dupBranchBody` at `python_options.go:257` (both branches identical — likely a real logic bug worth a second look).
- **`S1039` — `fmt.Sprintf` with no verbs** at `docker.go:125,128`.
- **`QF1012`** — `sb.WriteString(fmt.Sprintf(…))` at `api.go:87,100,103`; use `fmt.Fprintf(&sb, …)`.
- **Zero package documentation.** `revive` reports 50 missing-comment findings and there is no `doc.go` anywhere. Every exported symbol in a package that another package imports should have a comment starting with its own name.
- **`makefile` should be `Makefile`.** Universal Go/Unix convention; `make` finds both, humans and editors expect the capital.
- **`go.mod` declares `go 1.23.0` with `toolchain go1.24.2`.** Harmless but worth aligning; also pin the Go version in CI.
- **`.gitignore` has no trailing newline** and does not cover `*.exe`, `cli`, `*.sqlite3`, `.venv/`, `coverage.out`.
- **`vscode-test/` contains a full Django virtualenv** (hundreds of files) sitting untracked in the working tree. It is scratch space from manual testing; delete it and add `vscode-test/` to `.gitignore`.
- **`dist/` is 68 MB** on disk (correctly gitignored). `make clean` should remove it.

---

## 7. Structure, tooling and conventions

### 7.1 Folder structure — first principles

The current layout follows the `cmd/` + `internal/` convention correctly at the top level. The failure is one level down: `internal/ui` is not a package, it is a *namespace*. 40 files and 7,379 lines share one scope, so every function can call every other function, and the compiler enforces no boundary between "draw a progress bar" and "run `docker pull`".

The guiding principle from current Go practice — and the one this repo most needs — is **package by domain, not by layer**, with shallow hierarchies. The `ui` package should contain only what a terminal needs; the knowledge of how to build a Django project belongs somewhere that has never heard of Bubble Tea.

Proposed layout:

```
cmd/django-stack/          renamed from cmd/cli — binary name should match
internal/
  ├── config/              ProjectConfig, validation, feature flags   (no deps)
  ├── scaffold/            Plan/Step orchestration, the generators
  │   ├── python/          interpreter discovery, venv, pip/uv
  │   ├── django/          startproject, startapp, settings editing, migrations
  │   ├── docker/          compose generation, port allocation
  │   └── assets/          //go:embed templates (merges template + templates)
  ├── tui/                 renamed from ui — Model/Update/View ONLY
  └── errorsx/             renamed from errors — stops shadowing stdlib
test/e2e/                  black-box tests against the built binary
```

Two properties this buys you, both currently absent:

- `internal/config` and `internal/scaffold` import neither `bubbletea` nor `lipgloss`, so they are testable without a terminal. That alone would lift coverage past 60% with the tests that already exist.
- The dependency arrow points one way: `tui → scaffold → config`. Today it is a cycle-free accident rather than an enforced invariant.

Note also that `internal/errors` shadows the stdlib `errors` package; any file needing both must alias one. `errorsx` or `clierr` avoids this.

### 7.2 Testing

**State before this pass:** 52 tests, 5.9% coverage, 0 `t.Parallel()`, 0 benchmarks, 0 fuzz targets, no `-race` in any script, no CI.

Quality where tests exist is good — `TestGeneratedBookViewsUseGetObjectOr404ForStaleIDs` names a behaviour, not a function, and the seam-injection pattern (`SetPortListenerForTest`) is sound. The gap is coverage of the orchestration layer, and it is a *structural* gap: §5 M3 and §4 H5 explain why that code cannot currently be reached from a test.

Conventions to adopt:

- `t.Parallel()` on every leaf test — the suite is I/O-light and should run in well under a second.
- `-race` in CI, always. It is the only automated defence against C1.
- `t.TempDir()` over `os.MkdirTemp` (already used in 12 places — good).
- A `Fuzz` target on `ValidateProjectName` / `ProjectModuleName`; the hyphen→underscore transform crossing a Python-identifier check is exactly the shape fuzzing finds bugs in.
- Golden files for the 25 `generate*` functions. Asserting on `strings.Contains` of a 200-line Python file tests almost nothing; `testdata/*.golden` with a `-update` flag tests all of it.

### 7.3 Tooling gaps and drop-in configs

Nothing is enforced today: no CI, no `.golangci.yml`, no `.editorconfig`, no pre-commit. The irony is that this tool *generates* ruff, mypy and pre-commit configs for its users while having none itself.

**`.golangci.yml`** — start here; it makes §6 self-enforcing:

```yaml
version: "2"
linters:
  enable:
    - errcheck
    - govet
    - ineffassign
    - staticcheck
    - unused
    - errorlint      # catches the %v-instead-of-%w class
    - noctx          # catches the missing-context class
    - gosec
    - gocritic
    - revive
    - unparam
    - misspell
  settings:
    gocyclo:
      min-complexity: 30
  exclusions:
    rules:
      - path: _test\.go
        linters: [gosec]
formatters:
  enable: [gofmt, goimports]
```

**`.github/workflows/ci.yml`** — requires removing `.github/` from `.gitignore` first (C3):

```yaml
name: CI
on: [push, pull_request]
jobs:
  test:
    strategy:
      matrix:
        os: [ubuntu-latest, macos-latest, windows-latest]
    runs-on: ${{ matrix.os }}
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: '1.24', cache: true }
      - run: gofmt -l ./cmd ./internal | tee /dev/stderr | (! read)
      - run: go vet ./...
      - run: go test -race -coverprofile=coverage.out ./...
  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: '1.24' }
      - uses: golangci/golangci-lint-action@v6
```

The cross-OS matrix matters more here than in most projects: this codebase branches on `runtime.GOOS` in at least six places (`utils.go`, `python.go`, `scripts.go`, `venv.go`, plus the `install_unix.go`/`install_windows.go` build-tag pair) and none of the Windows paths have ever been exercised by an automated test.

**Also add:** `.editorconfig`; `make lint`, `make cover` and `make ci` targets; GoReleaser for cross-compiled release binaries (replacing `build-all.sh` and removing any reason to commit binaries).

### 7.4 Go conventions — where this codebase diverges

Measured against [Effective Go](https://go.dev/doc/effective_go), [Google's Go Style Decisions](https://google.github.io/styleguide/go/decisions) and [Best Practices](https://google.github.io/styleguide/go/best-practices):

| Convention | Status | Note |
|---|:---:|---|
| `gofmt` clean | ✗ | 5 files |
| Errors lowercase, no trailing punctuation | ✗ | 6 sites |
| Wrap with `%w` | ~ | Correct in `internal/`, absent in `cmd/` |
| `context.Context` as first param for I/O | ✗ | Created internally, never accepted |
| Accept interfaces, return structs | ✗ | `CLIError.Print()` hardcodes `os.Stderr`; should take `io.Writer` |
| Doc comments on exported symbols | ✗ | 50 revive findings, no `doc.go` |
| Package names: short, lowercase, no plurals | ✗ | `templates` vs `template`; `errors` shadows stdlib |
| Test files adjacent to code under test | ✗ | `test/ui/` + 194 LoC of export shims |
| Module path is a repo URL | ✗ | `module django-stack` |
| Zero value useful / no mandatory init | ✗ | `NewModel()` does network I/O |
| Table-driven tests | ~ | Only 3 `t.Run` across 52 tests |
| `internal/` for private code | ✓ | Correct |
| Build tags for platform files | ✓ | `install_unix.go` / `install_windows.go` |
| Version injected via `-ldflags` | ✓ | `makefile` does this properly |
| Sentinel errors as package vars | ✓ | Well designed — just unused |

### 7.5 Security triage

21 `gosec` findings. Most are false positives *in the current design*, but two deserve a decision:

- **G204 / G702 — subprocess with variable arguments** (`app.go:21`, `django.go:15`, `venv.go:22,29`, `python.go:86,89,140`, `docker.go:101,276`). All flow from `SelectedPythonPath`/`AppName`/`ProjectName`. `exec.Command` does not invoke a shell, and `ValidateProjectName`/`ValidateAppName` constrain the two user-supplied strings to `^[a-zA-Z][a-zA-Z0-9_-]*$`. **This is genuinely safe today** — but only because of those validators, which is worth an explanatory comment at each call site so a future change doesn't quietly remove the guarantee.
- **`cmd/cli/install_windows.go:39-40` — string-built PowerShell.** `installDir` is derived from `%LOCALAPPDATA%`, so it is not attacker-controlled in any normal scenario, but this is the one place a shell *is* involved and the pattern is concatenation. Prefer passing the value via an environment variable rather than interpolating into the script text.
- **G306 — `os.WriteFile(targetPath, input, 0755)`** at `install_unix.go:25`, writing to `/usr/local/bin`. `0755` is correct for an executable; suppress with a documented `//nolint:gosec` rather than leaving it in the noise.

Positives worth noting: secrets use `crypto/rand` (`docker.go`), `.env` is written `0600`, `.env.example` `0644`, and the generated Django settings apply real security headers. The security *output* of this tool is better than its own posture.

---

## 8. Changes made during this audit

Deliberately minimal — one real bug fix and test coverage. No refactors were performed; §9 sequences those for you to approve.

**Bug fixed — data race in `internal/logger`.** `log()` read `l.level` at line 95 *before* acquiring the mutex at line 99; `Success`, `Skip` and `DryRun` had the same inverted order. Concurrent `SetLevel` was therefore an unsynchronised read/write. The lock now precedes the level check in all four methods.

**Tests added — 33 new tests, all passing under `-race`:**

| File | Tests | Covers |
|---|---:|---|
| `internal/logger/logger_test.go` | 10 | Level filtering, format-arg handling, JSON validity and escaping, prefixes, status helpers, plus a concurrency test that fails under `-race` without the fix above |
| `internal/errors/errors_test.go` | 5 | `Error()` format, `errors.Is` compatibility, `WithHint` copy-not-mutate semantics, sentinel code uniqueness and hint completeness |
| `internal/ui/model_test.go` | 14 | `ValidateProjectName` / `ValidateAppName` (23 table cases incl. `../escape` and reserved words), `ProjectModuleName`, `CalculateTotalSteps` (incl. the shared ruff/pre-commit lint step), `ProcessFormData` label→flag mapping and stale-flag clearing, `GetActiveForm` routing, and concurrency of `AppendStepMessage` / `UpdateProgress` |
| `internal/template/loader_test.go` | 4 | That `//go:embed` actually resolved — `loadEmbeddedCSS` swallows its error, so a renamed asset would otherwise ship empty stylesheets silently |

Two of these are regression guards for latent bugs rather than characterisation tests: `TestCalculateTotalStepsIsAlwaysPositive` (a zero would make `UpdateProgress` divide by zero) and `TestProcessFormDataClearsStaleFlagsOnReRun` (backing up through the form must not accumulate flags).

**Measured effect:** 52 → 85 tests · coverage 5.9% → **12.6%** · `internal/logger` 0% → 85.7% · `internal/template` 0% → 100% · `internal/errors` 0% → 33.3% · `internal/ui` 6.4% → 9.9%.

The `internal/ui` number stays low for the structural reason in §4 H5: the orchestration core cannot be reached from a test until it is decoupled from the TUI.

---

## 9. Prioritised roadmap

**Now — hygiene, ~1 hour, no behaviour change**
1. Remove `.github/` from `.gitignore`; extend it with `*.exe`, `cli`, `*.sqlite3`, `.venv/`, `vscode-test/`, `coverage.out`. **(C3)**
2. `git rm --cached cli django-stack.exe db.sqlite3`; delete the `vscode-test/` venv from disk. **(H1)**
3. `gofmt -w ./cmd ./internal`. **(§6)**
4. Add `.golangci.yml` and `ci.yml`; make CI red until §6 is clean. **(§7.3)**
5. Rename `makefile` → `Makefile`; add `lint`, `cover`, `ci` targets.

**Next — correctness, ~1 day**
6. Fix the `View` mutation and guard `StepMessages` reads. **(C2, C1 interim)**
7. Fix the 8 dropped errors, starting with `multienv.go:84`. **(M1)**
8. Fix `%v`→`%w` in `cmd/`, lowercase the 6 error strings, delete the 3 unused vars and 4 dead params. **(§6)**
9. Set `module github.com/vaibdix/django-stack`. **(H2)**
10. Decide on `internal/errors` and `internal/logger`: wire in or delete. **(H4)**

**Then — structure, ~1 week**
11. Extract `internal/config` (`ProjectConfig` + validation, zero dependencies).
12. Extract `internal/scaffold` with the `Plan []Step` model; `CreateProject` becomes a loop over the plan and `CalculateTotalSteps` becomes `len(plan)`. **(H5)**
13. Thread one `context.Context` from `main` through generation; cancel on quit. **(M4, M5)**
14. Move generation off `*Model` entirely — config in, `tea.Msg` out. Delete both mutexes. **(C1, closed properly)**
15. Move `test/ui/ui_test.go` into `internal/ui/`; rename `test_exports.go` → `export_test.go`. **(H3)**
16. Move `NewModel`'s discovery I/O into `Init()` commands. **(M3)**

**Later — polish**
17. Convert the 25 `generate*` functions to `//go:embed` + `text/template` with golden-file tests. **(M7)**
18. Merge `internal/template` + `internal/templates` into `internal/assets`. **(M6)**
19. Type the feature flags so UI labels are free-text. **(M8)**
20. Package doc comments and a `doc.go` per package; `t.Parallel()` throughout; add GoReleaser and a `CONTRIBUTING.md`.

---

## 10. Addendum — findings from re-verification and remediation

A second full read-through of every file, plus the Phase 0/1 remediation work, produced five new
findings and one correction. Two were found *by* the new tests rather than by reading — which is the
argument for the tests.

### Correction to C1 — the race is narrower than stated

§3 C1 claimed roughly thirty unsynchronised fields. Re-verification of the actual write set shows:

- `ProgressStatus` is never written by the generator goroutine — it is mediated through
  `Program.Send(ProjectProgressMsg)` → `Update`, on the main goroutine.
- `TotalSteps` and the `ProjectConfig` fields are written *before* the goroutine starts
  (`update.go:146,168`), giving a happens-before edge.
- The docker port/password fields are written and read entirely within the goroutine's own call chain.

**`StepMessages` is the only live race**: written under `StepMessagesMu` by `AppendStepMessage`
(`model.go:205-209`), read *without* the lock by `View` (`view.go:101-109`).

The architectural fix still stands. The current arrangement is safe by accident, not by construction —
any future field written during generation and read in `View` silently becomes a race. But the
severity of what exists *today* was overstated, and the record should be accurate.

### A1 — `View` renders an empty progress log for the first three steps *(new, high)*

`view.go:103-107`:

```go
start := len(m.StepMessages)
if start > 3 {
    start = len(m.StepMessages) - 3
}
for _, msg := range m.StepMessages[start:] {
```

`start` is initialised to `len`, and the clamp only fires when `len > 3`. For 1, 2 or 3 messages
`StepMessages[start:]` is **empty** — the "Recent steps:" header renders with nothing beneath it.
Should be `start := 0`. No test covered `View`'s `StepSetup` branch, so this was invisible.

### A2 — the progress bar overshoots 100% in every configuration *(new, high)*

`CalculateTotalSteps` is the denominator; the count of `UpdateProgress` call sites is the numerator's
ceiling. They disagree in every case, measured by `TestProgressAccountingIsCurrentlyInconsistent`:

| Configuration | `CalculateTotalSteps()` | `UpdateProgress` calls | Bar reaches |
|---|---:|---:|---:|
| minimal | 3 | 6 | 200% |
| Tailwind only | 4 | 8 | 200% |
| everything enabled | 15 | 16 | 107% |

Three separate causes:

1. **Tailwind fires twice** — `creator.go:174` and `tailwind.go:20` both call `UpdateProgress`, while
   `StepCountTailwind` budgets one.
2. **Two call sites are unbudgeted** — `venv.go:161` ("Installing development dependencies") and
   `migrations.go:86` ("Creating sample data") have no corresponding `StepCount*` constant.
3. **The baseline is wrong** — the minimal path budgets 3 steps and fires 6.

This is the concrete cost of the 15 hand-maintained `StepCount*` constants (`model.go:126-142`) being a
parallel list that nothing forces to match the work actually performed. `len(Plan(cfg))` removes the
class of defect rather than fixing one instance.

### A3 — `ConfigureDjangoSettings` was not idempotent *(new, medium — fixed)*

`replaceManagedBlock` (`settings.go:246-251`) stripped every leading newline from the suffix and
re-joined with a single `\n`, while the append path in `InjectEnvConfig` (`settings.go:392`) emits
`\n\n`. Re-running the pipeline therefore shifted the remainder of `settings.py` up one line each
time. The scaffolder re-opens `settings.py` after other features have edited it, so this ran in
practice.

Found by `TestConfigureDjangoSettingsIsIdempotent` on its first execution. Fixed by restoring the
separator blank line; the test now guards it.

### A4 — six dead `Step` constants *(new, low)*

`StepFeatures`, `StepTemplates`, `StepCreateApp`, `StepAppTemplates`, `StepServerOption`,
`StepGitInit` (`types.go:13-19`) are never assigned and never matched in `Update`'s transition switch
or `View`'s render switch. They describe a state machine that does not exist.

### A5 — undocumented ordering hazard between docker setup and `.env` *(new, medium)*

`SetupDockerPostgres`/`SetupDockerMySQL` write `PostgresDockerPassword` (`docker.go:122-123,231-232`);
`renderEnvFileContent` reuses it *if already set*, otherwise generates a fresh one
(`env.go:161-163`). Correct only because of call order in `creator.go:84-111`. Reordering yields a
`.env` whose password does not match the running container — a failure that surfaces as an opaque
database connection error.

Unlike the multi-env and pre-commit constraints, which carry explanatory comments, this one is
undocumented. Now recorded in `canonicalStepOrder` (`internal/ui/creator_contract_test.go`) with an
enforcing assertion.

### Measured effect of Phase 0 + Phase 1

| Metric | Audit baseline | After Phase 0/1 |
|---|---:|---:|
| Tests | 52 | **107** |
| Golden files | 0 | **59** |
| Total coverage | 5.9% | **21.4%** |
| `internal/templates` | 0% | **79.0%** |
| `internal/logger` | 0% | **85.7%** |
| `internal/template` | 0% | **100%** |
| `internal/ui` | 6.4% | **17.5%** |
| `gofmt -l` | 5 files | **0** |
| CI | none, and impossible | 3-OS matrix + lint |

`internal/ui` remains low for the structural reason in §4 H5: the orchestration core cannot be reached
from a test until it is decoupled from the TUI. That is Phase 3.

---

## §11 — Final state (all phases complete)

Every phase of the remediation plan has landed. This section records where the
codebase ended up, and — more usefully — which findings turned out to be wrong
or incomplete once the work started.

### Measured

| Metric | Audit | Now |
|---|---|---|
| `golangci-lint` issues | 105 | **0** |
| Tests (incl. 3 fuzz targets) | 52 | **161** |
| Golden files | 0 | **71** |
| Total coverage | 12.6% | **44.9%** |
| `gofmt -l` | 5 files | **0** |
| Largest package | `internal/ui`, 7,379 LoC / 40 files | `internal/scaffold`, ~2,600 / 30 |
| Largest function | `CreateProject`, complexity 62 | none above 30 |
| Charm deps outside the TUI | pervasive | **0** |
| Generated content in Go string literals | ~1,900 lines | **0** (31 embedded templates) |
| Mutexes | 2, and one unguarded read | **0** |
| CI | none, and `.gitignore` made it impossible | 4 jobs, 3 OSes, lint gating |

### Findings the work corrected

The audit was wrong or incomplete in five places. Each was caught by a test
rather than by re-reading:

1. **F1 overstated.** The audit claimed ~30 unsynchronised fields. Only
   `StepMessages` was a live race; the rest were mediated by `Program.Send` or
   written before the goroutine started. The architecture fix was still right,
   for a different reason: the model was safe *by accident*, and any new field
   would have made it unsafe.
2. **Progress accounting was worse than reported.** Not just Tailwind — the
   declared total and the actual number of progress updates disagreed in
   *every* configuration. The bar reached 200% on a minimal project, because
   the TUI budgeted 3 steps for work that takes 12.
3. **`replaceManagedBlock` was not idempotent** — it emitted one fewer newline
   than the append path, so re-running the pipeline shifted `settings.py` up a
   line each time. Found by a golden, not by reading.
4. **Three dead parameters**, each proven dead by two goldens coming out
   byte-identical: `generateWindowsStartScript(useTailwind)`,
   `generateBookFormTemplate(cfg.HTMX)`, `generateBookDeleteTemplate(cfg.HTMX)`.
   Plus four generators taking an `appName` they never read.
5. **`needsIncludeInProjectURLs` ended in `|| true`** — unconditionally true
   while reading as a condition. Not in the audit at all.

### Judgment calls worth knowing about

- **`govet`'s `shadow` check is disabled.** It ran through five phases and found
  zero real bugs; every hit was `if err := f(); err != nil` inside a scope that
  already had an `err`, which is the idiom the Go team recommends. Keeping it
  meant either uglifying correct code or carrying permanent noise.
- **Five `gosec` rules are excluded**, each with its reasoning recorded inline in
  `.golangci.yml`. Writing files and executing a user-selected interpreter is
  what this program *is*; taint analysis cannot see that. The one property that
  exclusion could have hidden — `.env` staying `0600` — is asserted by
  `TestSecretFilesAreNotWorldReadable` instead.
- **`render` panics rather than returning an error.** The templates are embedded
  and the data types are compile-time constants of the package, so a failure is
  a broken binary, not a user error. `TestEveryTemplateParsesAndRenders` makes
  that a build failure rather than a user-facing crash.
- **Templates use `[[ ]]`, not `{{ }}`.** Most generated content is Django
  template source, which is itself built from `{{ }}` and `{% %}`.

### What is now enforced rather than intended

- `go list -deps` proves the engine imports no terminal code — a CI job, not a
  convention.
- Regenerating the goldens on a clean tree must be a no-op; a diff fails CI.
- The step order, including the three load-bearing constraints, is asserted
  against `plan()` — the payoff for the Phase 0 characterisation test that
  recorded that order before anything was touched.
- The progress bar's denominator and the number of updates it receives are the
  same list, so they cannot drift.

---

## §12 — What running CI locally found

The workflows were written but never executed. Running them with
[nektos/act](https://github.com/nektos/act) against OrbStack, before any push,
found **four bugs that `make ci` could not** — every one of them would have
failed on the first real CI run, and one would have shipped a broken repository.

### 1. `.gitignore` excluded the entire CLI source

`.gitignore` listed the built binary as `django-stack`. Unanchored, that pattern
matches any file **or directory** of that name at any depth — including
`cmd/django-stack/`, the CLI's source directory.

```
$ git check-ignore -v cmd/django-stack/main.go
.gitignore:4:django-stack	cmd/django-stack/main.go
```

A `git add -A` would have committed a repository with no `cmd/` directory at
all. This is the same failure shape as the original F24 (`.gitignore` containing
`.github/`, making CI impossible to commit) — reintroduced while fixing it.
The binary patterns are now anchored (`/django-stack`), which matches the root
artifact and not the source tree.

Nothing local catches this: the working tree is complete, `go build` passes, and
the tests pass. It is only visible to something that reconstructs the repo from
what git would actually store.

### 2. The golden check silently did nothing

The workflow ran `go test ./internal/... -run TestGolden -update`. Only two
packages define an `-update` flag, so the other six failed with *"flag provided
but not defined"* — the step had never once compared a golden.

I had also run this command locally with `2>&1 >/dev/null`, which hid the
failure from me. Fixed by targeting the packages that own goldens, and moved
into `make goldens-check` so it is covered locally too.

Two further holes surfaced while fixing it:

- `-run TestGolden` did not match `TestComposeFileGoldens`, so the compose
  goldens were excluded. The pattern is now `-run Golden`.
- `git diff` ignores untracked files, so on a tree where goldens had never been
  committed the check passed vacuously. It now uses `git status --porcelain`.
- `internal/scaffold/dbdocker` had golden files but **no `-update` flag** — its
  goldens could be compared but never regenerated. It has one now.

### 3. `golangci-lint-action@v6` cannot run golangci-lint v2

```
Error: invalid version string 'v2.12', golangci-lint v2 is not supported by
golangci-lint-action v6, you must update to golangci-lint-action v7.
```

`.golangci.yml` is on the v2 schema. The action is now v7, with the linter
version pinned exactly (`v2.12.2`) rather than to a range — the config uses
gosec's `G702`/`G703`, which do not exist in 2.1.x and are rejected outright by
`config verify` there. Pinning keeps CI on the same schema as local.

### 4. `.goreleaser.yaml` used a field that does not exist

```
yaml: unmarshal errors:
  line 46: field checksums not found in type config.Project
```

GoReleaser v2 spells it `checksum`, singular. After the fix, `goreleaser build
--snapshot` produces all six targets and `-version` reports the injected
metadata, so the `-X main.version` ldflag is confirmed wired end to end.

### Limits of local CI

`act` is necessary but not sufficient:

- **It only runs Linux containers.** The `macos-latest` and `windows-latest`
  matrix legs cannot run — and those are the reason the matrix exists, since
  the scaffolder branches on `runtime.GOOS` in interpreter discovery, venv
  paths, start scripts and `--install`. Those paths remain unexercised until a
  real push.

  **Correction (2026-08-11).** This section originally said those legs were
  *skipped*. They were not. With no `-P` mapping for them, act resolved an empty
  image name and the whole job died during setup:

  ```
  [CI/test (ubuntu-latest)-1 ]   ❌  Failure - Set up job
  Error: invalid reference format
  ```

  So `act push` never ran the `test` job at all — the only job that runs the
  suite under `-race`. This is the same shape as the two defects §12 was written
  to record: a check that reads as though it runs and does not. `.actrc` now
  pins `--matrix os:ubuntu-latest`, and the job passes.

  Note also that act does not read the docker context. On OrbStack it needs
  `DOCKER_HOST` exported explicitly, or it reports *"Couldn't get a valid docker
  connection"* while the `docker` CLI works normally.
- **It shares one host directory across jobs**, where GitHub gives each job its
  own checkout. Run concurrently, the architecture job rewrites goldens while
  the test job reads them. `.actrc` sets `--concurrent-jobs 1` so local runs do
  not fail for a reason that cannot happen on GitHub.
- **The runner image is not GitHub's.** `catthehacker/ubuntu` is close, but a
  failure that depends on preinstalled tooling can still differ.

---

## References

- [Effective Go](https://go.dev/doc/effective_go)
- [Google Go Style — Decisions](https://google.github.io/styleguide/go/decisions) · [Best Practices](https://google.github.io/styleguide/go/best-practices)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [golangci-lint linters](https://golangci-lint.run/docs/linters/)
- [golang-standards/project-layout](https://github.com/golang-standards/project-layout) — widely referenced, explicitly not official
- [How to structure Go projects — practical conventions, not rules](https://daveamit.com/posts/2026-02-13-go-folder-structure/)
- [Go Project Structure: Practices & Patterns](https://www.glukhov.org/post/2025/12/go-project-structure/)
