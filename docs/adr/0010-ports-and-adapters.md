# 10. The engine reaches the machine through ports

Status: accepted, 2026-10-27

## Context

The executor touched the machine in several ways, each with its own seam:

- a `command.Runner` for programs;
- a `clock` field for the time;
- a `deps` struct with a hash lookup and an entropy reader;
- `os` calls for the project folder;
- `dbdocker` functions that took the runner;
- `crypto/rand` and `net.Listen` inside `dbdocker` for the database password and port.

Tests replaced some of these and not others. Nothing proved that a fake behaved like the real thing it stood in for.

## Decision

- `internal/port` holds the interfaces and nothing else:
  - `Runner`;
  - `FS`: stat, read, list, make and remove folders, and `WriteIn(root, rel, …)`, which refuses paths that leave the root;
  - `Index`: the sha256 hashes of a release;
  - `Docker`: ready, start, stop and a free local port.

  `port.Ports` bundles them with a `Clock` and a `Rand` reader. A depguard rule keeps `os`, `net`, `command`, `dbdocker`, `pypi` and `adapter` out of the package.
- `internal/adapter` holds the real implementations:
  - `OSFS`, which writes through `os.Root`;
  - `PyPI`;
  - `DockerCLI`, over `dbdocker`;
  - `Complete`, which fills unset ports with them, `command.NewOS`, `time.Now` and `crypto/rand`.

  `command.Runner` is now an alias of `port.Runner`.
- `internal/port/porttest` holds the fakes: an in-memory `MemFS`, a map `Index`, a recording `Docker`, a fixed `Clock` and a seeded `Rand`. The existing `commandtest.Fake` is the runner fake.
- Contract tests in `internal/adapter` run the same checks against each real adapter and its fake:
  - **files:** round trip, file modes, escape refusal, name order, removal;
  - **index:** sorted hashes, unknown version, cancellation, against an `httptest` index;
  - **runner:** output, and cancellation as a `*command.Error`;
  - **Docker:** a cancelled start, plus a real start and stop under `DJANGO_STACK_LIVE`;
  - **free port and randomness:** a usable port, fresh bytes on each read, and the same seed giving the same bytes.
- `Env` holds one `port.Ports`. `scaffold.Options.Ports` and `scaffold.Preflight(…, ports)` replace the runner and clock options. The secret key, the database passwords and the database port all come from the ports, so a run with a fixed clock, a seeded `Rand` and the fake Docker port is fully determined by its spec.

## Consequences

- A fake that drifts from its adapter fails a contract test, not a later live run.
- The purity test also forbids `adapter` from `project` methods.
- Generated projects are unchanged.
