#!/usr/bin/env bash
# Mirror internal/cli/active-apis.json MCP → datpaq/cli (reverse direction).
#
# Prefer regenerating in the CLI repo first:
#   cd ../CLI && make fetch-active-apis && make sync-active-apis
# That copies CLI → MCP. Use this script only when you edited the
# manifest here by mistake and need to push the fix back to CLI.
#
# Usage: ./scripts/sync-active-apis.sh [path-to-cli-repo]
# Default target: ../CLI (sibling layout)
#
# Copies the file to the CLI repo working tree but does NOT commit
# there — review and commit in the CLI repo manually.

set -euo pipefail

ROOT="$(git rev-parse --show-toplevel)"
SOURCE="$ROOT/internal/cli/active-apis.json"
TARGET_ROOT="${1:-$ROOT/../CLI}"
TARGET="$TARGET_ROOT/internal/cli/active-apis.json"

if [[ ! -f "$SOURCE" ]]; then
  echo "error: source not found: $SOURCE" >&2
  exit 1
fi

if [[ ! -d "$TARGET_ROOT" ]]; then
  echo "error: target repo not found: $TARGET_ROOT" >&2
  echo "       pass the path as the first argument" >&2
  exit 1
fi

if [[ ! -f "$TARGET" ]]; then
  echo "error: target file does not exist: $TARGET" >&2
  exit 1
fi

if cmp -s "$SOURCE" "$TARGET"; then
  echo "already in sync."
  exit 0
fi

cp "$SOURCE" "$TARGET"
echo "synced: $SOURCE"
echo "    -> $TARGET"
echo ""
echo "Next: cd $TARGET_ROOT && git add internal/cli/active-apis.json && git commit"
