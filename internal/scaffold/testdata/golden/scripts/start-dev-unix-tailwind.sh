#!/bin/sh
set -eu

SCRIPT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
cd "$SCRIPT_DIR"

"/usr/local/bin/code" -n "$SCRIPT_DIR" >/dev/null 2>&1 &
echo "VS Code opened in a new window."
echo "The workspace auto-runs the 'Workspace: Start Dev' task in integrated terminals."
echo "Watch the Django and Tailwind terminals inside VS Code."
echo "Open http://localhost:8000 once the server is ready."
