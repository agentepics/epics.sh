#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
SOURCE_PATH="${1:-$REPO_ROOT/../agentepics/footer.md}"
TARGET_PATH="$REPO_ROOT/internal/epic/footer.md"

if [ ! -f "$SOURCE_PATH" ]; then
  echo "missing footer source: $SOURCE_PATH" >&2
  echo "pass a path explicitly or place the canonical repo at ../agentepics" >&2
  exit 1
fi

cp "$SOURCE_PATH" "$TARGET_PATH"
echo "synced canonical footer to $TARGET_PATH"
