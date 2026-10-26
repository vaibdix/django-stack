# tools/matrix

The phase sweep (gate G in `finalplan.md`). It generates the Tier 2 project set with a `-race` build of the CLI, runs every project's own toolchain against it, drives the pages in Chromium, and diffs each project against the baseline sweep.

Requirements: `pict` (`brew install pict`), `uv`, Docker (OrbStack), and Playwright for the web UI checks:

```sh
cd tools/matrix/ui && npm ci && npx playwright install chromium
```

A sweep, from the repository root:

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
