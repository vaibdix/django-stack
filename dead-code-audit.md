# django-stack — Dead Code Audit

Date: 2026-08-12
Commit: 550e1e9 (branch `feature/refactor_scaffold_engine`)
Scope: all Go source, embedded templates, static assets, and golden files.

---

## 1. Summary

The codebase has no dead templates, no dead CSS, and no dead enum
constants. `golangci-lint` reports the `unused` linter clean. The dead code
that exists is invisible to that linter, because the linter counts a
reference from a test as a use.

Six groups of dead code exist. One is a whole package. One is a
configuration field that is always false, which makes a pipeline step and
two branches unreachable.

| ID | Finding | Severity | Prod. lines |
|---|---|---|---|
| D1 | `internal/clierr` is an orphan package | High | 152 |
| D2 | `Project.RunServer` is always false, so one plan step is unreachable | High | ~18 |
| D3 | Test-only code lives in production files | Medium | ~163 |
| D4 | Five symbols have zero references | Medium | ~23 |
| D5 | One golden file is an orphan | Low | 0 |
| D6 | Three `doc.go` files hold no documentation | Low | 3 |

Total: about 359 lines of production Go, plus 152 lines of test code and
one golden file of 23 lines.

### Scorecard

| Category | Result |
|---|---|
| Dead packages | 1 of 8 |
| Unreachable from `main` (module only) | 22 functions |
| Genuinely unreferenced symbols | 5 |
| Test-only symbols in production files | 13 |
| Dead templates | 0 of 49 |
| Dead CSS files | 0 of 4 |
| Dead `Step` constants | 0 of 12 |
| Orphan golden files | 1 of 113 |

---

## 2. Method

Four passes ran against the tree. Each pass finds a class the others miss.

1. `deadcode ./cmd/django-stack`, filtered to the module path. This gives
   reachability from `main`, so test files do not hide dead production code.
2. `golangci-lint run -E unused,unparam`. This counts tests as uses, so it
   is the control group for pass 1.
3. A reference sweep over every exported and unexported top-level symbol.
   Each symbol got two counts: references from non-test files, and
   references from `_test.go` files. The split separates "dead" from
   "test-only".
4. A golden-file sweep. The command `make goldens` ran, and file
   modification times were compared before and after. A golden file with an
   unchanged time is written by no test.

Passes 1 and 3 agree on every finding below.

---

## 3. Findings

### D1 — `internal/clierr` is an orphan package (152 lines)

No file outside the package imports it:

```
$ grep -rn "internal/clierr" --include='*.go' .
internal/clierr/clierr_test.go:9: clierrors "github.com/vaibdix/django-cli/internal/clierr"
```

The only importer is its own test. The package does not link into the
binary:

```
$ go list -deps ./cmd/django-stack | grep django-cli
.../internal/assets
.../internal/config
.../internal/scaffold/command
.../internal/scaffold/dbdocker
.../internal/scaffold/pyenv
.../internal/scaffold
.../internal/tui
.../cmd/django-stack
```

`internal/clierr` is absent from that list. The package holds 14 error
sentinels with codes E001 to E014, each with hint text. The generators build
errors with `fmt.Errorf` instead.

This is the unresolved half of finding H4 in `audit.md`. That finding said
"either wire `internal/errors` into the generators, or remove both packages.
Leaving them is the worst option." The git history shows what happened:

```
2613be8 refactor: drop the errors package superseded by clierr
496f386 refactor(clierr): add a CLI error taxonomy that renders to an io.Writer
```

The package was renamed and given a 152-line test suite. It was never wired
in. The test suite tests dead code.

Two build files name the dead package in their package list:

- `Makefile:59` — the `arch` target
- `.github/workflows/ci.yml:70` — the `architecture` job

Both run `go list -deps ... ./internal/clierr/...` to make sure that no
engine package depends on Bubble Tea. The check passes, because the package
depends on almost nothing.

**Decision needed.** The hint text is good CLI output, and
`CLIError.Fprint` is what the failure screen in `internal/tui/view.go:345`
needs. Wire it in, or remove all 304 lines. The current state is the third
option that `audit.md` already rejected.

---

### D2 — `Project.RunServer` is always false (high)

`config.Project` has two fields for one concept:

```go
RunServer          bool   // project.go:62
StartDevServer     bool   // project.go:76
```

`internal/tui.Model` embeds `config.Project`, so both fields sit on the same
struct at runtime.

`StartDevServer` is live. The form at `internal/tui/startup.go:58` writes it,
and `update.go:204` and `view.go:322` read it.

`RunServer` is written once in production, at `internal/tui/startup.go:35`:

```go
RunServer:          false,
```

No other production line writes it. Only two test files set it true. As a
result, three pieces of code are unreachable in a shipped binary:

| Location | Dead item |
|---|---|
| `internal/scaffold/plan.go:95` | The step `report server instructions`, gated by `when: wantsServer` |
| `internal/scaffold/plan.go:183` | `(*Env).reportServerInstructionsStep` |
| `internal/scaffold/vscode.go:270` | The body of `(*Env).reportServerInstructions` |

The guard is also duplicated. The plan gates the step on `wantsServer`, and
then the function body tests the same field again:

```go
func (e *Env) reportServerInstructions() {
	if e.Cfg.RunServer {
```

The lost behavior is real. A user who asks for a development server gets no
printed instruction for the manual start command, the start script, or the
Tailwind watch command. The text exists and never reaches the screen.

**Two corrections are possible.** Set `RunServer` from the same form that
sets `StartDevServer`, and remove the duplicate guard. Or remove
`RunServer`, its plan step, and both functions.

---

### D3 — Test-only code lives in production files (medium, ~163 lines)

The following symbols have zero references from non-test files. Each lives
in a file that is not named `_test.go`, so each compiles into the shipped
binary.

| Symbol | Location | Lines | Test refs |
|---|---|---|---|
| `command.Fake`, `NewFake`, 6 methods, `Call.String` | `scaffold/command/fake.go` (whole file) | 101 | 26 |
| `scaffold.NopReporter`, `scaffold.RecordingReporter` | `scaffold/scaffold.go:16-30` | 15 | 28 |
| `dbdocker.Stop` | `scaffold/dbdocker/dbdocker.go:216` | 24 | 4 |
| `NodeManager.Dlx`, `NodeManager.DlxCommand` | `scaffold/nodepm.go:51,57` | 10 | 6 |
| `SettingsEditor.Content`, `SettingsEditor.SetContent` | `scaffold/settings.go:32,36` | 8 | 44 |
| `assets.AllAuthLogin`, `assets.AllAuthSignup` | `assets/allauth_templates.go:174-175` | 2 | 6 |
| `generateBookAPI` | `scaffold/api_generators.go:41` | 3 | 7 |

This is finding H3 in `audit.md` in a new form. The old form was
`internal/ui/test_exports.go` with 29 `…ForTest` functions. That file is
gone. The repository now has the correct pattern in two places,
`internal/scaffold/export_test.go` and
`internal/scaffold/dbdocker/export_test.go`, but the seven items above did
not follow it.

Three of the items need separate comment.

**`command.Fake`** is used by 8 test files across 4 packages. A rename to
`export_test.go` cannot work, because a `_test.go` file is not visible to
another package. Move the type to a new package,
`internal/scaffold/command/commandtest`, or put a build tag on `fake.go`.

**`dbdocker.Stop`** stops and removes a project container. Nothing calls it.
The TUI has no teardown path, so a container that a failed run started stays
running. This reads as a missing feature, not as dead code.

**`generateBookAPI`** hardcodes an application name:

```go
func generateBookAPI(pydantic bool) string {
	return render("api_views.py.tmpl", apiPackageData{AppName: "catalog", Pydantic: pydantic})
}
```

The production path is `generateAPIPackage` at `api_generators.go:22`, which
passes the real name. `generateBookAPI` is a test fixture with a generator
name, and 7 tests call it. A reader cannot tell from the name that the
value "catalog" never reaches a generated project.

---

### D4 — Five symbols have zero references (medium)

These symbols are referenced by no file, test or otherwise.

| Symbol | Location | Note |
|---|---|---|
| `scaffold.NewEnv` | `scaffold/scaffold.go:47` | The only constructor for `Env`. The TUI builds `&scaffold.Env{...}` at `tui/generate.go:38`, and every test builds a literal too. |
| `command.Look` | `scaffold/command/runner.go:62` | A byte-identical copy of `pyenv.CommandAvailable` at `scaffold/pyenv/venv.go:28`. The `pyenv` copy has 4 callers. |
| `scaffold.StaticDir` | `scaffold/constants.go:5` | The only unused constant in the file. Callers write the literal `"static"`. |
| `tui.ProjectProgressMsg` | `tui/types.go:20` | Superseded by `scaffoldProgressMsg` in `tui/generate.go:15`. |
| `tui.ProjectCreationDoneMsg` | `tui/types.go:25` | Superseded by `scaffoldDoneMsg` in `tui/generate.go:17`. |

`NewEnv` is the item worth attention. A package that exports a constructor
that nobody calls has two ways to build the same value. The literal path
skips the `Runner` and `Reporter` defaults that `NewEnv` sets. The `Env`
methods handle a nil `Runner` at `scaffold.go:76` and a nil `Reporter` at
`scaffold.go:57`, so the defaults are safe to lose.

The duplicate pair `Look` and `CommandAvailable` breaks Rule "one item, one
name". Keep `pyenv.CommandAvailable`.

---

### D5 — One golden file is an orphan (low)

`internal/scaffold/testdata/golden/onboarding/SUPERUSER_SETUP.md` (23 lines)
is read by no test and written by no test.

Proof: `make goldens` regenerates every golden file. After that command, the
modification time of every golden file changed, except this one file.

The head commit explains it:

```
550e1e9 test(scaffold): gate the htmx code with strict mypy and fold the
        superuser guide into the README
```

The generator for `SUPERUSER_SETUP.md` was removed. The golden file stayed.

The `goldens-check` target cannot find this. That target runs
`git status --porcelain` on the golden paths, which reports files that
changed or that are untracked. An orphan golden is committed and unchanged,
so it stays invisible.

**Correction:** remove the file and its directory. To find this class in
future, make `goldens-check` also fail on a golden file that the `-update`
run did not write.

---

### D6 — Three `doc.go` files hold no documentation (low)

```
internal/assets/doc.go        →  package assets
internal/scaffold/pyenv/doc.go →  package pyenv
internal/tui/doc.go            →  package tui
```

Each file is one line. A `doc.go` file exists to hold a package comment.
`CLAUDE.md` forbids comments in this repository, so these three files can
never hold one. They add three files and one package clause each.

---

## 4. What is clean

These classes were tested and found clean. They are listed so that a later
audit does not repeat the work.

- **Templates.** All 49 files under `internal/scaffold/templates/` are
  reachable. The 12 `book_*.html.tmpl` files look unreferenced to a
  literal-string search, because `AppTemplateConfig.templateFor` at
  `api_templates.go:16` builds each name at runtime.
- **CSS.** All 4 files under `internal/assets/templ/css/` are embedded,
  written to disk, and linked from the base template.
- **Enum constants.** All 12 `tui.Step` constants have between 8 and 25
  references. Finding A4 in `audit.md` reported six dead `Step` constants.
  That finding is corrected.
- **Struct fields.** Every field of `config.Project` and `tui.Model` is
  read, except `RunServer` (see D2).
- **Golden directories.** 22 of 23 directories are live. Only `onboarding`
  is an orphan.
- **`golangci-lint`.** The `unused` linter reports no findings. The
  `unparam` linter reports 2 findings, both in test helpers:
  `conf_runtime_test.go:54` and `golden_test.go:209`.

---

## 5. Removal plan

Do the steps in this order. Each step keeps the tree green.

1. Remove `internal/scaffold/testdata/golden/onboarding/`. Then run
   `make goldens-check`.
2. Remove the three `doc.go` files listed in D6. Then run `go build ./...`.
3. Remove `scaffold.NewEnv`, `command.Look`, `scaffold.StaticDir`,
   `tui.ProjectProgressMsg`, and `tui.ProjectCreationDoneMsg`.
4. Decide D2. If the instructions must reach the user, set `RunServer` from
   the development-server form. If they must not, remove the field, the plan
   step, and both functions.
5. Decide D1. If the error taxonomy must be used, wire `clierr` into the
   generators and the failure screen. If it must not, remove the package,
   its test, and the two package lists in `Makefile:59` and
   `ci.yml:70`.
6. Move `command.Fake` to a `commandtest` package. Then move the other D3
   items into the `export_test.go` file of their own package.
7. Rename `generateBookAPI` to show that it is a fixture, or give it an
   `appName` parameter and call it from `generateAPIPackage`.

After every step, run `make ci`.

---

## 6. Why the current checks missed this

The repository has strong checks. Each of the six findings escaped for a
specific reason, and each reason is worth recording.

| Finding | Why it escaped |
|---|---|
| D1 | `unused` counts a test as a use. The package has a full test suite. |
| D2 | A field that is always false is valid Go. No linter reads it as dead. |
| D3 | Same reason as D1. Test-only code is used code to a linter. |
| D4 | `unused` does not report exported symbols in a package that other packages import. |
| D5 | `goldens-check` finds changed and untracked files, not orphan files. |
| D6 | An empty `doc.go` file is valid Go. |

One check closes four of the six. Add `deadcode` to the `architecture` job:

```
go install golang.org/x/tools/cmd/deadcode@latest
deadcode -filter='github.com/vaibdix/django-cli' ./cmd/django-stack
```

That command reports D1, D3, and D4 today. Add an allowlist for the
test-only symbols that stay, and the job then fails on the next symbol that
becomes unreachable.
