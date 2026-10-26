# 11. Apps, middleware, settings blocks and routes declare where they go

Status: accepted, 2026-10-27. Supersedes the last decision of ADR 4.

## Context

ADR 4 removed the string patching from settings rendering but kept the order implied:

- a feature's apps went into `apps` or `lateApps`, its blocks into `blocks` or `lateBlocks`;
- inside a slot, the order was the order of `features()`;
- URL patterns were appended in the same order.

Moving a feature in the registry would have reordered `INSTALLED_APPS` and `MIDDLEWARE` in every generated project, and nothing would have noticed two features putting the same thing in the same place.

## Decision

- An app or middleware a feature adds is an `entry{name, after, rank}`:
  - `after` names the entry it follows (`django.contrib.staticfiles`, `XFrameOptionsMiddleware`, `allauth`, `rest_framework`), or `atEnd`;
  - entries that follow the same name are ordered by `rank`.
- `placeEntries` places them into the base list: Django's apps or middleware, then the shared apps and the project app. It is a function of the entries, not of their order. It refuses:
  - a name listed twice;
  - two entries claiming the same place;
  - an anchor that is not in the list.
- Settings blocks are `block{section, rank, text}`. The section is before or after `AUTH_USER_MODEL`. `orderBlocks` sorts them and refuses two blocks in one place. The engine's own blocks hold ranks 1 to 5 (passwords, login, the CSP report, the single-file environment, console logging). Features use the shared ranks in `model.go`:

  | Rank | Feature |
  |---|---|
  | 10 | CSP |
  | 20 | HTMX |
  | 30 | AllAuth |
  | 40 | structlog |
  | 50 | Sentry |
  | 60 | mypy |
  | 70 | the API frameworks |
- URL routes are `route{rank, line}`, sorted by rank:

  | Rank | Route |
  |---|---|
  | 10 | home |
  | 20 | admin |
  | 30 | health |
  | 40 | Django auth |
  | 41 | AllAuth |
  | 50 | the API routes |
  | 60 | the app |

  Code blocks before `urlpatterns` are ranked blocks too.
- The ranks were chosen so that every configuration renders exactly as before: the goldens, the Tier 1 fingerprint and the six real projects are unchanged.
- A test renders settings and URLs for every stack, both API frameworks and both CSP paths with every add-on that contributes an entry, and fails on any clash.

## Consequences

- The order of `features()` no longer decides the order of apps, middleware, blocks or routes.
- A feature that needs a new place takes a new rank or a real anchor. A clash fails at render time with both names.
- `renderSettingsModule`, `settingsFiles`, `urlFiles` and `buildProjectURLsContent` return an error.
