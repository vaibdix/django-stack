#!/bin/sh

set -eu

ROOT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
cd "$ROOT_DIR"

echo "Manual smoke checklist for django-stack"
echo

DEFAULT_PYTHON="$(command -v python3 || command -v python || true)"
if [ -n "$DEFAULT_PYTHON" ]; then
  echo "System default Python:"
  "$DEFAULT_PYTHON" --version 2>/dev/null || true
  echo "  $DEFAULT_PYTHON"
  echo
fi

echo "Other Python interpreters found on PATH:"
if command -v which >/dev/null 2>&1; then
  which -a python3 python 2>/dev/null | awk '!seen[$0]++' | while IFS= read -r py; do
    [ -n "$py" ] || continue
    version="$("$py" --version 2>/dev/null || true)"
    printf "  %s -> %s\n" "$py" "$version"
  done
else
  echo "  'which' not available on this system"
fi
echo

echo "Next steps:"
echo "  1. Run: GOCACHE=/tmp/django-stack-gocache GOMODCACHE=/tmp/django-stack-gomodcache go run ./cmd/cli"
echo "  2. Confirm the Python picker preselects the system default shown above."
echo "  3. Confirm the Django picker shows the latest compatible stable releases for that Python."
echo "  4. Complete a sample project setup and run '.venv/bin/python manage.py check' (or Windows equivalent)."
