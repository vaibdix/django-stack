# Contributing

## Getting set up

```
git clone https://github.com/vaibdix/django-cli.git
cd django-cli
make ci        # fmt-check, vet, test -race, lint — everything CI runs
```

You need Go 1.26+. Docker is optional: only the database container tests use it,
and they are skipped when it is unavailable.

## Running CI locally

`make ci` runs the same checks CI runs, but on your machine, with your toolchain.
To run the actual workflow in the actual containers, use [nektos/act](https://github.com/nektos/act):

```
brew install act
make ci-act    # runs .github/workflows/ci.yml: architecture, test, e2e, lint
```

Four things are worth knowing before you trust a green local run:

- **Use `make ci-act`, not `act push`.** A bare `act push` also fires
  `release.yml`, which fails with `git doesn't contain any tags`. That is a local
  artefact — the release workflow only runs on tags upstream — but it looks like
  a CI failure. `ci-act` passes `-W .github/workflows/ci.yml` to avoid it.
- **On OrbStack, `DOCKER_HOST` must be set.** act does not read the docker
  context, so it looks for `/var/run/docker.sock` and reports "Couldn't get a
  valid docker connection" even though the `docker` CLI works. `ci-act` sets it
  for you; export your own value if you use something other than OrbStack.
- **Only the Linux leg of the test matrix runs.** act runs Linux containers, so
  `macos-latest` and `windows-latest` cannot run locally — and those are the
  reason the matrix exists, since the scaffolder branches on `runtime.GOOS` in
  interpreter discovery, venv paths, start scripts and `--install`.
- **The runner image is not GitHub's.** `catthehacker/ubuntu` is close, but a
  failure that depends on preinstalled tooling can still differ.

act runs against your working tree, not a clean checkout, so it sees untracked
files. That is a feature: it is how a stray `cmd/` scratch package gets caught
before it reaches a PR.

## The one rule that matters

**Only `internal/tui` may import Bubble Tea.**

Everything else — `config`, `scaffold` and its subpackages, `assets`, `clierr` —
must stay terminal-free, because that is what makes them testable at all. The
check is mechanical:

```
go list -deps ./internal/scaffold/... | grep charmbracelet   # must print nothing
```

The dependency arrow points one way:

```
tui -> scaffold -> {pyenv, dbdocker, command, assets} -> config
```

A leaf package that needs something from a parent declares its own narrow
interface (see `pyenv.Reporter` and `dbdocker.Reporter`) rather than importing
upward.

## Golden files

Generated output is pinned byte-for-byte in `testdata/golden/`. If a change is
meant to alter what a scaffolded project looks like:

```
go test ./internal/scaffold -run TestGolden -update
git diff internal/scaffold/testdata/golden
```

Then read the diff. **A golden diff nobody can explain is a regression** — that
is the whole point of the files existing. They are what let the package
restructure and the `text/template` migration be verified as behaviour
preserving rather than hoped to be.

## Generated content lives in templates

Python, HTML, YAML and shell that ends up in a user's project belongs in
`internal/scaffold/templates/`, not in a Go string literal. Templates use
`[[ ]]` delimiters, not Go's default `{{ }}`, because most of them are Django
templates that are themselves full of `{{ }}` and `{% %}`.

Add a field to a template and `TestEveryTemplateParsesAndRenders` will tell you
if you forgot to add it to the data struct.

## Naming

One verb per role. This is enforced by review, not by a linter:

| Prefix | Meaning | Shape |
|---|---|---|
| `render*` | pure content generation | `(cfg) string` — no I/O, no error |
| `write*` | writes exactly one file | `(env, path) error` |
| `apply*` | multi-file feature installation | `(env, path) error` |
| `new*` | constructor | `New*(…) (*T, error)` |
| `find*` / `detect*` | discovery, may fail | `(…) (T, bool)` |
| `is*` / `has*` | predicate | `(…) bool` |

Export only what is used outside the package. Most of the scaffolding engine is
unexported; `Env.Generate` is the entry point.

## Tests

- Prefer `t.Parallel()`. Do not use it in a test that calls `t.Setenv` or one of
  the `with*` helpers that swap a package-level variable — both mutate shared
  state and the race detector will say so.
- Tests live in-package. There is no `test_exports.go`; if a test needs an
  unexported identifier, it can already reach it.
- `go test -race ./...` must pass. The TUI is single-writer by construction and
  that is a property worth keeping.

## Releasing

Tag and push; GoReleaser builds the matrix and drafts the release.

```
git tag -a v1.2.3 -m "v1.2.3"
git push origin v1.2.3
```

Never commit built binaries.
