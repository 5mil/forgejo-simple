#!/usr/bin/env bash
# apply-minimal-install.sh
# Run this inside a full Forgejo source tree (v16.x) to apply the first simplification.
# This is a helper script — the real patches still need to be written against the actual source.

set -euo pipefail

echo "=== forgejo-simple: applying minimal install changes ==="

if [ ! -f "go.mod" ] || ! grep -q forgejo go.mod 2>/dev/null; then
  echo "Error: this does not look like a Forgejo source tree."
  echo "Clone upstream first:"
  echo "  git clone --depth 1 --branch v16.0.3 https://codeberg.org/forgejo/forgejo.git"
  exit 1
fi

echo "Source tree detected."
echo ""
echo "Manual steps still required (see patches/0001-sqlite-only-minimal-install.md):"
echo "  1. Edit the install template to show only Title + Admin fields"
echo "  2. Force db_type = sqlite3 in the install handler"
echo "  3. Generate a minimal app.ini"
echo ""
echo "After editing, build with:"
echo "  TAGS=\"bindata timetzdata sqlite sqlite_unlock_notify\" make build"
echo ""
echo "This script is a placeholder until the real unified diff is ready."
