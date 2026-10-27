# 20. make targets name the test tiers, and live tests allocate ports and names from one helper

Status: accepted, 2026-10-27

## Context

new.md asks for these things:

- `make` targets as the only interface: `check`, `content-check`, `ci`, `live` and `sweep`.
- Visual baselines that only `make ui-baseline` writes.
- Live tests that take their port and project name from a single helper.
- Sweep run folders outside the repo, with a size budget, that are generated and cleaned in batches by default.

The repo had `content-check` and `ci`, and the live tier as `test-live`. Each Docker live test picked its own port and name:

- `AvailablePort` with a fixed preferred port;
- a project name from `time.Now().UnixNano()`.

The sweep's batches of 20 were a shell loop kept outside the repo.

## Decision

- **Make targets:**
  - **`make check`:** the fast tier. It runs `fmt-check`, `vet`, `goldens-check` and `go test -short ./...`, in about 15 seconds.
  - **`make live`:** the live tier, the same as `test-live`.
  - **`make sweep RUN=<folder> [BASELINE=<run>]`:** runs `go run ./tools/matrix sweep`. With `BASELINE`, the screenshots are compared with that run's and are never written into it.
  - **`make ui-baseline RUN=<folder>`:** a sweep labelled `baseline` that records its own screenshots. Only this target makes a run that other sweeps compare with.

  Both sweep targets first install the UI check's packages (`npm ci` and Playwright's Chromium) in `tools/matrix/ui`.
- **`matrix sweep`:**
  1. It selects a new run folder if none exists. The selection refuses a folder inside the repository.
  2. It builds the binary if it is missing.
  3. It works through the projects in batches (`-batch 20`). Each batch is generated, checked and cleaned before the next one starts.
  4. After each batch, it measures the run folder. It stops when the folder holds more than `-max-gb` (20 GiB by default).
  5. It writes the report.

  Generate and check run with `-resume`, so after a stop the same command continues where it stopped.
- **`livetest.Port(t)`:** asks the kernel for an ephemeral port and checks that the port is also free on every interface. It records each port it hands out, so two tests in one process never get the same one.
- **`livetest.Name(t, prefix)`:** returns `prefix-<pid>-<n>`. That name is unique across the parallel test processes that `go test ./...` starts.
- The compose live test and the Docker adapter contract test use both helpers.

## Consequences

- The three Docker live runs (Postgres, MySQL and the adapter lifecycle) pass while they run at the same time in two packages.
- A sweep is one command that bounds its own disk use. A stopped sweep resumes, and it never writes screenshots into the baseline it compares with.
- `test-live` stays, because the README and older ADRs name it.
