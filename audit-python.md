# django-stack — Generated Python Audit

**Date:** 2026-08-11
**Commit audited:** `3cf69e8` (branch `feature/refactor_scaffold_engine`)
**Scope:** The Django project this tool *writes* — 11 Python templates, the settings/pyproject/dotenv
generators in `internal/scaffold`, and the dependency and tooling configuration that ships with a
generated project. The Go code itself is audited separately in [`audit.md`](audit.md).
**Reviewer stance:** Staff-engineer review against Django, DRF, uv and Ruff practice current as of
August 2026.

---

## 1. Executive summary

`audit.md` graded the Go. Nothing has ever graded its output. That output is the product; the Go is
the delivery mechanism.

The generated project works. It boots, it migrates, the CRUD demo functions, the security header
block is better than most tutorials, and the multi-env split is correctly ordered. What it is not is
**reproducible, current, or safe by default.**

Three findings stand out:

1. **A generated project cannot be reinstalled.** `pyproject.toml` receives `[project]
   name/version/requires-python` and nothing else — and it is only written at all when Ruff or Mypy
   is selected. Thirteen-plus packages are installed imperatively into `.venv` and recorded nowhere.
   No `requirements.txt`, no `uv.lock`, no `.python-version`. The virtualenv is the only record of
   what the project needs. Delete it and the project is unrecoverable.
2. **The API accepts anonymous writes.** `DEFAULT_PERMISSION_CLASSES` is `AllowAny`, the serializer
   is `fields = '__all__'`, and there is no throttling. A scaffold that generates a public writable
   endpoint by default teaches the wrong lesson to the person using it.
3. **The tool installs with uv but generates a project uv cannot manage.** `uv venv` and `uv pip
   install` are used for speed, and then nothing uv-shaped is left behind. This is the gap between
   "fast to create" and "possible to maintain".

None of these need a rewrite. §5 sequences the fixes.

### Scorecard

| Dimension | Grade | One-line verdict |
|---|:---:|---|
| Reproducibility | **F** | No manifest, no lock; the venv is the only record of what the project needs |
| Dependency hygiene | **D** | psycopg2 for every project incl. SQLite; two libraries doing one job; hook pins ~2 years stale |
| API security posture | **D** | Anonymous writes enabled by default, `fields = '__all__'`, no throttling |
| Settings / security | **C** | Solid header and hasher block, but no CSP, no `STORAGES`, `SECRET_KEY` fails open |
| Django idiom | **C** | Works and reads cleanly; no model `Meta`, ordering repeated 5×, URLs built by concatenation |
| Type safety | **C−** | Mypy is offered as a feature over Python with zero annotations |
| Tests generated | **F** | `self.assertTrue(True)` |
| Tooling config | **C+** | Ruff selection predates its current defaults; `known-third-party` hand-maintained |

### Verified dependency baseline (PyPI, 2026-08-11)

| Package | Scaffold ships | Current |
|---|---|---|
| django | 6.1 ✅ | 6.1 |
| **psycopg2-binary** | always installed | **psycopg 3.3.4** |
| djangorestframework | unpinned | 3.18.0 |
| drf-spectacular | unpinned | 0.30.0 |
| django-allauth | unpinned | 65.19.0 |
| django-environ | unpinned | 0.14.0 |
| django-htmx | unpinned | 1.29.0 |
| django-browser-reload | unpinned | 1.21.0 |
| django-stubs / drf-stubs | unpinned | 6.0.9 / 3.17.1 |
| ruff / djlint | unpinned | 0.16.2 / 1.44.2 |
| pre-commit-hooks hook | `v4.6.0` | **`v6.0.0`** |
| ruff-pre-commit hook | `v0.8.0` | **`v0.16.2`** |
| djLint hook | `v1.36.1` | **`v1.44.2`** |
| django-upgrade hook | `1.21.0` | **`1.31.1`** |

Django 6.1 declares `requires-python >=3.12`. That number is load-bearing — see C2.

---

## 2. Critical findings

### C1 — The generated project has no dependency manifest

`pyproject.go:24-28` writes exactly this, and only this:

```toml
[project]
name = "…"
version = "0.1.0"
requires-python = "…"
```

It is written only when Ruff or Mypy is chosen — `ruff.go:10` and `mypy.go:26` are the sole callers
of `NewPyprojectEditor`. Decline both features and the generated project has no `pyproject.toml` at
all.

Meanwhile packages arrive imperatively and are recorded nowhere:

| Where | Installs |
|---|---|
| `exec_bridge.go:38-39` | django, django-browser-reload, django-environ, python-dotenv, psycopg2-binary, argon2-cffi |
| `plan.go` (docker step) | mysqlclient |
| `api.go:16` | djangorestframework, drf-spectacular |
| `allauth.go:15` | django-allauth |
| `htmx.go` | django-htmx |
| `mypy.go:12-19` | mypy, django-stubs, djangorestframework-stubs |
| `precommit.go:15` | pre-commit |
| `formatting.go:27-28` | ruff, djlint |

There is no `requirements.txt`, no `uv.lock`, no `.python-version`. Clone the repository the tool
just initialised onto another machine and there is no supported way to install what it needs.

**Impact:** the project is not reproducible, not CI-buildable, and not deployable without a human
reconstructing the dependency list by reading `INSTALLED_APPS`.

### C2 — `requires-python` can contradict the pinned Django

`python_tooling.go:17` defaults the floor to `3.10` and otherwise derives it from the *selected
interpreter*:

```go
major, minor := 3, 10
if selectedMajor, selectedMinor, _, _, ok := pyenv.ParseDetailedVersionOutput(…); ok { … }
```

Django 6.1 requires `>=3.12`. Select a 3.10 or 3.11 interpreter with a 6.x Django and the manifest
declares `requires-python = ">=3.10"` next to `django==6.1` — an unsatisfiable resolution.

This is inert today only because nothing ever resolves from `pyproject.toml`. It becomes a hard
`uv lock` failure the moment C1 is fixed.

The Django→Python floor table exists, but in the wrong place: `internal/tui/django_versions.go:44-51`
(`djangoCompatibilityRules`). `internal/scaffold` cannot import it — tui→scaffold is the dependency
direction, and `make arch` forbids charm dependencies inside scaffold. `internal/config` is the valid
shared home; both packages already import it and it sits inside the arch fence.

---

## 3. High-severity findings

### H1 — psycopg2-binary is installed into every project, including SQLite-only ones

`exec_bridge.go:39` installs `PkgPsycopg` unconditionally. Three problems compound:

- It is installed for SQLite and MySQL projects that will never import it.
- psycopg **2** is legacy. psycopg 3 is what new projects should use, and Django ≥4.2 supports it
  with no `ENGINE` change.
- psycopg2-binary wheels historically lag new CPython releases, and the version picker offers
  Python 3.14 (`django_versions.go:45-47`).

`internal/assets/postgres_readme.go:14-17` also instructs the user to `pip install psycopg2-binary`.

### H2 — The generated API accepts anonymous writes

`api.go:36-38`:

```python
'DEFAULT_PERMISSION_CLASSES': [
    'rest_framework.permissions.AllowAny',
],
```

Compounded by `serializers.py.tmpl` using `fields = '__all__'` (every column exposed, including any
added later without review), no `DEFAULT_THROTTLE_CLASSES`, and `BrowsableAPIRenderer`
(`api.go:41`) enabled in every environment including production. `BookViewSet` has no filter, search
or ordering backends.

### H3 — pre-commit revisions are roughly two years stale

`templates/config/pre-commit-config.yaml.tmpl` pins `v4.6.0`, `v0.8.0`, `v1.36.1` and `1.21.0`
against current `v6.0.0`, `v0.16.2`, `v1.44.2` and `1.31.1`. Roughly two years of Ruff rules and
django-upgrade fixers never run. Hard-coding revisions in a Go template guarantees this recurs.

### H4 — python-dotenv and django-environ both ship, doing one job

`exec_bridge.go:39` installs both. `settings.go:383-387` imports both. `settings.go:360-363` calls
`load_dotenv(BASE_DIR / '.env')` and then constructs `environ.Env()`. django-environ's
`environ.Env.read_env()` replaces the pair, removing a dependency and an import.

### H5 — `DJANGO_SETTINGS_MODULE` written into `.env` is dead configuration

`multienv.go:60-61` appends `DJANGO_SETTINGS_MODULE=<module>.settings.local` to `.env`. It can never
take effect: `.env` is read from *inside* the settings module (`settings.go:360`), by which point
Django has already resolved which settings module to import.

What actually works is the separate rewrite of `manage.py`, `wsgi.py` and `asgi.py`
(`multienv.go:52-58`). The `.env` entry is inert, and misleading — editing it appears to switch
environments and does nothing.

---

## 4. Medium and low findings

| ID | Finding |
|---|---|
| **M1** | Zero type annotations in generated Python, while Mypy is offered as a feature |
| **M2** | `models.py.tmpl` has no `Meta` — no `ordering`, `indexes` or `constraints`. Consequence: `.order_by('-created_at')` repeated five times in `views.py.tmpl` |
| **M3** | `sample_data.py.tmpl` assigns float literals (`39.99`) to a `DecimalField`; should be `Decimal("39.99")`. Not wrapped in `transaction.atomic` |
| **M4** | `tests.py.tmpl` ships `self.assertTrue(True)` — a test that cannot fail |
| **M5** | Ruff config (`ruff.go:18-49`) predates current practice: `select` replaces Ruff's defaults (now `F, E, B, UP, RUF`), so `ignore = ["RUF012"]` is inert. Missing `DTZ`, `PTH`, `RET`, `TC`, `N`, `ARG`, `LOG`. `S101` blanket-ignored rather than scoped to tests. `known-third-party` is a hand-maintained list that will drift |
| **M6** | No `SECURE_CSP`. Django 6.0 shipped native CSP (`SECURE_CSP`, `SECURE_CSP_REPORT_ONLY`, `django.middleware.csp.ContentSecurityPolicyMiddleware`); the injected security block predates it |
| **M7** | No `STORAGES`. `ManifestStaticFilesStorage` under `STORAGES["staticfiles"]` is the 4.2+ shape; `production.py.tmpl` is its natural home |
| **M8** | `SECRET_KEY` fails open (`settings.go:368`): a working default means a production boot with the variable missing silently uses a publicly known key |
| **M9** | `generateSecretKey` (`dotenv.go:138-146`) has modulo bias — `chars[int(b)%len(chars)]` over a 50-character alphabet and 256 byte values |
| **M10** | `ACCOUNT_EMAIL_VERIFICATION = 'none'` (`allauth.go:53`) is a development default shipped as the only default |
| **M11** | ruff and djlint are installed into every `.venv` by the mandatory formatting step but appear in no manifest — invisible dev dependencies |
| **M12** | `views.py.tmpl` mixes function-based HTML CRUD and a DRF `ViewSet` in one module while `api.py` holds only the router |
| **L1** | `<int:book_id>` where Django convention is `<int:pk>` |
| **L2** | `redirect('/' + app_name_var + '/')` instead of `reverse()` / namespaced redirect |
| **L3** | Compose pins `postgres:16`, `mysql:8.0`; PostgreSQL 18 is current |
| **L4** | `book_list` renders every row — no pagination |
| **L5** | `isbn` has no validator; `price` has no `MinValueValidator` |
| **L6** | `admin_plain.py.tmpl` is a bare import plus `# noqa: F401` — an `admin.py` that registers nothing |
| **L7** | No `.python-version` file |
| **L8** | Templates are written single-quoted and `ruff format` rewrites them to double quotes on every generation |

---

## 5. Prioritised roadmap

| Phase | Addresses | Theme |
|---|---|---|
| 1 | C2 → C1, M11, L7 | Make the generated project reproducible |
| 2 | H1, H3, H4 | Dependency correctness |
| 3 | H2, H5, M6–M10 | Security posture |
| 4 | M1–M5, M12, L1–L8 | Python craft |

**Phase 1 must start with C2.** Fixing the manifest without fixing the Python floor produces a
`pyproject.toml` that `uv lock` rejects.

Dependencies are accumulated in Go during the run and written once, rather than calling `uv add` per
feature — each `uv add` re-resolves and re-locks, and there are up to nine install sites. Declare
with `>=` floors; `uv.lock` is what pins exactly.

The manifest step must land after every step that installs a package and before `run migrations`,
which constrains it to just after `setup multi-env` in `plan()`. It cannot go after
`run scaffold formatting` — that step runs post-migration. ruff and djlint are statically known
(`scaffoldFormattingPackages()`), so they enter the dev group without waiting for that step.

---

## 6. Explicitly not recommended

- **Do not swap mypy for Astral `ty`.** ty is in beta as of August 2026 with a stable 1.0 targeted
  later in the year, and first-class Django/django-stubs support is named as remaining beta→stable
  work. Keep mypy + django-stubs.
- **Do not replace DRF with django-ninja.** Out of scope for an audit of what we already generate.
- **Do not adopt pytest as the runner in this pass.** `tests.py` should gain real tests in Django's
  own `TestCase`; pytest-django is a separate proposal.
- **Do not add async views, `django.tasks`, `GeneratedField` or `db_default`.** All defensible, none
  prompted by these findings.

---

## 7. How these findings were verified

Dependency currency was checked against the PyPI JSON API on 2026-08-11, not from memory. Django
6.1's `requires-python` was read from the same source, which is what makes C2 concrete rather than
theoretical. Django 6.0/6.1 settings claims (`SECURE_CSP`, `STORAGES`, `SECRET_KEY_FALLBACKS`) were
checked against the Django 6.1 settings reference; the allauth setting names against the current
allauth configuration docs; the Ruff default rule set against the Ruff rules index; and ty's release
status against Astral's own beta announcement.

Everything in §2–§4 cites a file and line in this repository. No finding is inferred from what the
generated project *ought* to contain.

---

## 8. Remediation outcome

All four phases were implemented. Verified against a real generated project (Django 6.1, Python
3.13.13, uv 0.11.29, SQLite, REST + HTMX + AllAuth + Ruff):

| Check | Result |
|---|---|
| `rm -rf .venv && uv sync --frozen` | reinstalls from `uv.lock` alone |
| `manage.py check` | no issues |
| `manage.py test` | 5 generated tests, all pass |
| `POST /api/v1/books/` anonymous | **403** (was 201) |
| `GET /api/v1/books/` anonymous | 200, paginated |
| `/`, `/catalog/`, `/accounts/login/`, `/admin/`, `/api/schema/swagger-ui/` | all 200 |
| HTMX search + `hx-get/post/swap/target/trigger/confirm` | intact |
| Inline `<style>` blocks in served HTML | **0** (was 3 templates' worth) |
| `Content-Security-Policy-Report-Only` header | present |
| Minimal SQLite project, no features | `pyproject.toml` written; psycopg **not** installed |

Two decisions departed from the plan, both for reasons found during implementation:

**CSP ships report-only, not enforced.** The policy itself is strict (`default-src 'none'`) and
every page this scaffold generates complies — the three inline `<style>` blocks were moved into
`static/css/app.css`. But drf-spectacular's Swagger UI template inlines its bootstrap script, so
enforcing would break `/api/schema/swagger-ui/`. The generated settings carry a comment saying
exactly what to rename to enforce it.

**`ARG` was dropped from the Ruff rule set.** It fired three times on framework-mandated
signatures — `Command.handle(self, *args, **options)` and a DRF `@action(self, request)`. A rule
that cannot be satisfied without `noqa` on ordinary Django code is the wrong rule to select.

### A defect found in the verification itself

`TestGeneratedPythonSurvivesRuff` ran `ruff check --fix`, so it passed whenever a violation was
*autofixable* — which is most of them. Confirmed by mutation: adding an unused `import os` to
`models.py.tmpl` left the test green. Switched to `--no-fix`, which immediately surfaced two real
template defects the suite had never reported: three `force-single-line` violations, and a missing
blank line before a class in `serializers.py.tmpl`. The same test now also generates its Ruff config
by calling `applyRuff`, rather than keeping a hand-copied duplicate that could drift from `ruff.go`.
