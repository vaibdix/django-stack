# Dead Code Cleanup Record

Date: 2026-08-13
Branch: `cleanup`
Source audit commit: `550e1e9`
Cleanup base commit: `6485ddf`

## Result

The cleanup removed all confirmed dead production code from the source audit.
The application call graph now contains no unreachable module function.

| ID | Finding | Result |
|---|---|---|
| D1 | The `internal/clierr` package had no application importer. | Removed the package, its tests, and its architecture-list entries. |
| D2 | `Project.RunServer` was always false in production. | Removed the field, the plan step, the wrapper, and the reporting function. |
| D3 | Test helpers were in production source files. | Moved required helpers to test files or the `commandtest` package. Removed unused tests and helpers. |
| D4 | Five symbols had no references. | Removed all five symbols. |
| D5 | One golden file had no producer or reader. | Removed `onboarding/SUPERUSER_SETUP.md`. |
| D6 | Three `doc.go` files contained only package declarations. | Removed all three files. |

## Cleanup Details

### D1: Orphan Package

The cleanup removed `internal/clierr` because the application did not import it.
The cleanup also removed the package from the architecture checks.

### D2: Unreachable Server Report

The cleanup removed `Project.RunServer` and the unreachable server-reporting step.
The TUI and the generated README still show the development server commands.

The cleanup also removed `devServerCommand` after the D2 removal made that wrapper unreachable.
The VS Code generator still uses `devServerCommandFor`.

### D3: Test Helpers

The command fake now belongs to `internal/scaffold/command/commandtest`.
Tests import this package directly.

The reporter, settings-editor, Node-manager, and API helpers now exist only in test files.
AllAuth tests now use `AuthFormTemplate` and `AuthPages` directly.

The cleanup removed `dbdocker.Stop` and its two tests.
The application had no container teardown path that called this function.

### D4: Unreferenced Symbols

The cleanup removed these symbols:

- `scaffold.NewEnv`
- `command.Look`
- `scaffold.StaticDir`
- `tui.ProjectProgressMsg`
- `tui.ProjectCreationDoneMsg`

### D5: Orphan Golden File

The cleanup removed one 23-line golden file.
The scaffold golden count decreased from 60 files to 59 files.

### D6: Redundant Files

The cleanup removed these files:

- `internal/assets/doc.go`
- `internal/scaffold/pyenv/doc.go`
- `internal/tui/doc.go`

## Corrected Audit Facts

The original audit contained these incorrect statements:

- Test-only functions did not enter the shipped binary. The Go linker removed them.
- The repository had 112 golden files before cleanup, not 113.
- The golden files occupied 22 directories before cleanup, not 23.
- The application had 53 live scaffold templates, not 49.
- The `deadcode` command did not detect the orphan `clierr` package.
- The same command did not detect fields, constants, types, golden files, or redundant files.

## Verification

These checks pass on the uncommitted cleanup tree:

- `go build ./...`
- `go test -run '^$' ./...`
- `deadcode -filter='github.com/vaibdix/django-cli' ./cmd/django-stack`
- `make fmt-check vet arch test-race test-e2e lint`

The `deadcode` command prints no finding.
The linter prints `0 issues`.

The full `make ci` command stops at `goldens-check` because the orphan golden file is an uncommitted deletion.
This result is expected until a later commit records that deletion.

The `act` workflow has the same expected result.
Its test, lint, and end-to-end jobs pass.
Its architecture job stops only at the uncommitted golden-file deletion.

The final CLI binary is `dist/dead/code/django-stack-darwin-arm64`.
Its SHA-256 value is `01fd7fff8b91ed02da3765ca2956a13201a582f417ca68fe260eff026b9033da`.
