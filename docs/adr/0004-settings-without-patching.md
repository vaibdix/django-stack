# 4. Settings and URLs are rendered, not patched

Status: accepted, 2026-10-27. Its last decision (no anchor model) is superseded by ADR 11.

## Context

Two generated modules were fixed up after rendering:

- With multi-env settings, `base.py` was rendered as a single-file `settings.py`. Four string edits then removed the `ALLOWED_HOSTS` line, collapsed every run of blank lines to one, put back the two blank lines PEP 8 wants around `env_bool`, and added a `.parent` to `BASE_DIR`.
- The project `urls.py` held `MODULE.views` placeholders that a final replace turned into the project's module.

The plan proposed a typed settings model in which apps, middleware and blocks carry anchors (`After`, `Before`, `Order`) and one sort places them. No feature needs an anchor: every contribution goes into a fixed slot (`apps` or `lateApps`, `blocks` or `lateBlocks`), and the slots are already typed in `settingsContribution`.

## Decision

- The settings template takes `SettingsPackage`. It adds the third `.parent` and keeps one blank line, not two, between the sections it owns.
- `envConfig.SettingsPackage` leaves `ALLOWED_HOSTS` out of the env block, since `production.py` sets it.
- The tail of the settings module is written by a `sectionWriter`. In a settings package it caps the blank lines before each section at one. This is the one formatting rule left, and it is named.
- `urls.py` writes the module name into the error handlers directly.
- No anchor-based ordering model is added until a feature needs to place an app or middleware relative to another.

## Consequences

- No generated file is edited after rendering.
- The Tier 1 fingerprint (532,224 configurations) and the six real projects are unchanged, so the new rendering reproduces every old blank line.
