# tools/matrix

The full sweep. It generates the Tier 2 project set with a `-race` build of the CLI, runs every project's own toolchain against it, drives the pages in Chromium, and diffs each project against the baseline sweep.

Requirements: `pict` (`brew install pict`), `uv`, Docker (OrbStack), and Playwright for the web UI checks:

```sh
cd tools/matrix/ui && npm ci && npx playwright install chromium
```

A sweep, from the repository root. `make ui-baseline` records a baseline and `make sweep` compares with one; both install the Playwright dependencies first:

```sh
make ui-baseline RUN=~/django-stack-runs/baseline
make sweep RUN=~/django-stack-runs/next BASELINE=~/django-stack-runs/baseline
```

Both run `matrix sweep`. It selects the run folder if it is new and builds the binary. It then generates, checks and cleans the projects in batches of 20 (`-batch`), and stops if the folder grows past `-max-gb` (20) after a batch. Finally it writes the report. Run the same command again to resume a stopped sweep.

The same steps by hand:

```sh
go run ./tools/matrix select -label baseline
R=~/django-stack-runs/baseline-<sha>
go run ./tools/matrix build -run $R
go run ./tools/matrix generate -run $R
go run ./tools/matrix check -run $R
go run ./tools/matrix report -run $R
go run ./tools/matrix clean -run $R
```

A later phase reuses the baseline's selection, dates and versions, and compares against it:

```sh
go run ./tools/matrix select -label phase0 -like $BASELINE
go run ./tools/matrix check -run $R -ui-baseline $BASELINE
go run ./tools/matrix diff -run $R -baseline $BASELINE
```

Run output never goes inside the repository: `select` refuses a folder in any git work tree.

Tier 1 renders every configuration from the blueprint in memory, keeps one copy of each distinct file, and checks those copies: JSON and YAML parse, `py_compile`, Ruff's syntax rules, djlint's lint, and `docker compose config` on the compose files.

```sh
go run ./tools/matrix render-all
go run ./tools/matrix render-all -check=false
```

`reference` checks the committed [`reference/`](../../reference) project in a temporary copy. It runs `uv sync`, `ruff format --check`, `ruff check`, `mypy`, `djlint`, `manage.py check`, `makemigrations --check` and the project's tests. `make content-check` runs it after `render-all`.

```sh
go run ./tools/matrix reference
```
