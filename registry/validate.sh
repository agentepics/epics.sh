#!/usr/bin/env bash
set -euo pipefail

# Registry Epic Validator
# Validates registry/epics/*.json against the published source at
# github.com/agentepics/epics.
#
# Requires: gh (GitHub CLI, authenticated), uv, jq

REPO="agentepics/epics"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REGISTRY_DIR="$SCRIPT_DIR/epics"
TMPDIR_V=$(mktemp -d)
trap 'rm -rf "$TMPDIR_V"' EXIT

red()    { printf '\033[0;31m%s\033[0m\n' "$*"; }
dim()    { printf '\033[0;90m%s\033[0m\n' "$*"; }

# ── Preflight ──────────────────────────────────────────────────────

for cmd in gh uv jq; do
  if ! command -v "$cmd" &>/dev/null; then
    red "$cmd not found."; exit 1
  fi
done

# ── Fetch repo tree once ───────────────────────────────────────────

dim "Fetching tree from $REPO ..."
TREE_FILE="$TMPDIR_V/tree.json"
gh api "repos/$REPO/git/trees/main?recursive=1" > "$TREE_FILE"

# Fetch SKILL.md and EPIC.md blobs into individual files
dim "Fetching SKILL.md and EPIC.md blobs ..."
BLOBS_DIR="$TMPDIR_V/blobs"
mkdir -p "$BLOBS_DIR"

blob_entries=$(jq -r '
  .tree[]
  | select(.type == "blob")
  | select(.path | test("^[^/]+/(SKILL|EPIC)\\.md$"))
  | "\(.sha) \(.path)"
' "$TREE_FILE")

while read -r sha path; do
  safe_name=$(echo "$path" | tr '/' '_')
  gh api "repos/$REPO/git/blobs/$sha" --jq '.content' | base64 -d > "$BLOBS_DIR/$safe_name"
done <<< "$blob_entries"

# Build JSON map from fetched files
BLOBS_FILE="$TMPDIR_V/blobs.json"
(
  echo "{"
  first=true
  for f in "$BLOBS_DIR"/*; do
    fname=$(basename "$f")
    # Restore original path: agent-heartbeat_SKILL.md -> agent-heartbeat/SKILL.md
    orig_path=$(echo "$fname" | sed 's/_/\//')
    if [ "$first" = true ]; then first=false; else echo ","; fi
    printf '%s: %s' "$(printf '%s' "$orig_path" | jq -Rs .)" "$(jq -Rs . < "$f")"
  done
  echo ""
  echo "}"
) | jq . > "$BLOBS_FILE"

# ── Run validation ─────────────────────────────────────────────────

exec uv run --quiet --script - "$REGISTRY_DIR" "$TREE_FILE" "$BLOBS_FILE" <<'PYTHON'
# /// script
# requires-python = ">=3.10"
# dependencies = ["pyyaml"]
# ///
import json
import re
import sys
from pathlib import Path

import yaml

registry_dir = Path(sys.argv[1])
tree_json = json.loads(Path(sys.argv[2]).read_text())
blobs = json.loads(Path(sys.argv[3]).read_text())

errors = 0
warnings = 0
checked = 0

RED = "\033[0;31m"
YELLOW = "\033[0;33m"
GREEN = "\033[0;32m"
DIM = "\033[0;90m"
RESET = "\033[0m"


def error(msg: str) -> None:
    global errors
    print(f"{RED}  ERROR: {msg}{RESET}")
    errors += 1


def warn(msg: str) -> None:
    global warnings
    print(f"{YELLOW}  WARN:  {msg}{RESET}")
    warnings += 1


def ok(msg: str) -> None:
    print(f"{DIM}  ok:    {msg}{RESET}")


# Build lookups from the tree
tree_entries = tree_json["tree"]
tree_shas: dict[str, str] = {}
remote_dirs: set[str] = set()

for entry in tree_entries:
    p = entry["path"]
    if entry["type"] == "tree" and "/" not in p:
        tree_shas[p] = entry["sha"]
        remote_dirs.add(p)


def get_blob(path: str) -> str | None:
    return blobs.get(path)


def strip_frontmatter(text: str) -> str:
    return re.sub(r"^---\n.*?\n---\n*", "", text, flags=re.DOTALL)


def strip_h1(text: str) -> str:
    lines = text.strip().splitlines()
    if lines and lines[0].startswith("# "):
        lines = lines[1:]
    return "\n".join(lines).strip()


def parse_frontmatter(text: str) -> dict | None:
    m = re.match(r"^---\n(.*?)\n---", text, re.DOTALL)
    if not m:
        return None
    return yaml.safe_load(m.group(1))


# ── Required fields ────────────────────────────────────────────────

REQUIRED_FIELDS = [
    "slug", "title", "summary", "description", "category", "tags",
    "source", "version", "digest", "updatedAt", "validationStatus",
    "maintainers", "skillMd", "epicMd", "features",
]

VALID_STATUSES = {"reviewed", "draft", "pending"}

# ── Validate each registry file ───────────────────────────────────

for registry_file in sorted(registry_dir.glob("*.json")):
    reg = json.loads(registry_file.read_text())
    slug = reg.get("slug", registry_file.stem)
    checked += 1

    print(f"\n── {slug} ({registry_file.name})")

    # 1. Required fields
    for field in REQUIRED_FIELDS:
        if not reg.get(field):
            error(f"missing required field: {field}")

    # 2. Format checks
    if not re.match(r"^[a-z][a-z0-9-]*[a-z0-9]$", slug):
        error(f"slug '{slug}' is not valid kebab-case")

    if f"{slug}.json" != registry_file.name:
        error(f"slug '{slug}' does not match filename '{registry_file.name}'")

    version = reg.get("version", "")
    if not re.match(r"^\d+\.\d+\.\d+", version):
        error(f"version '{version}' is not semver")

    updated_at = reg.get("updatedAt", "")
    if not re.match(r"^\d{4}-\d{2}-\d{2}", updated_at):
        error(f"updatedAt '{updated_at}' is not a valid date")

    vs = reg.get("validationStatus", "")
    if vs not in VALID_STATUSES:
        warn(f"unknown validationStatus: '{vs}'")

    if not reg.get("tags"):
        warn("tags array is empty")

    if not reg.get("features"):
        warn("features array is empty")

    maintainers = reg.get("maintainers", [])
    if not maintainers:
        error("maintainers array is empty")
    elif not maintainers[0].get("name"):
        error("first maintainer has no name")

    source_repo = reg.get("source", {}).get("repo", "")
    source_path = reg.get("source", {}).get("path", "")
    if not source_repo:
        error("source.repo is missing")
    if not source_path:
        error("source.path is missing")

    # 3. Source exists in remote repo
    if source_path not in remote_dirs:
        error(f"source.path '{source_path}' not found in remote repo")
        continue
    else:
        ok("source path exists in remote")

    # 4. Digest (git tree SHA)
    remote_sha = tree_shas.get(source_path)
    local_digest = reg.get("digest", "")
    if not remote_sha:
        error(f"could not fetch tree SHA for '{source_path}'")
    elif local_digest == remote_sha:
        ok("digest matches tree SHA")
    else:
        error(f"digest mismatch: registry='{local_digest}' remote='{remote_sha}'")

    # 5. SKILL.md content match
    remote_skill = get_blob(f"{source_path}/SKILL.md")
    if remote_skill is None:
        error(f"SKILL.md not found in remote at {source_path}/")
    else:
        registry_skill = reg.get("skillMd", "")
        if remote_skill.strip() == registry_skill.strip():
            ok("skillMd matches remote SKILL.md")
        else:
            error("skillMd does not match remote SKILL.md")

    # 6. EPIC.md content match
    remote_epic = get_blob(f"{source_path}/EPIC.md")
    if remote_epic is None:
        error(f"EPIC.md not found in remote at {source_path}/")
    else:
        registry_epic = reg.get("epicMd", "")
        remote_body = strip_h1(strip_frontmatter(remote_epic))
        local_body = strip_h1(registry_epic)
        if remote_body == local_body:
            ok("epicMd body matches remote EPIC.md")
        else:
            error("epicMd body does not match remote EPIC.md")

    # 7. EPIC.md frontmatter cross-check
    if remote_epic:
        fm = parse_frontmatter(remote_epic)
        if fm is None:
            warn("no frontmatter in remote EPIC.md")
        else:
            if fm.get("id") != slug:
                error(
                    f"EPIC.md frontmatter id='{fm.get('id')}' "
                    f"!= registry slug='{slug}'"
                )
            remote_tags = sorted(fm.get("tags", []))
            registry_tags = sorted(reg.get("tags", []))
            if remote_tags != registry_tags:
                error(
                    f"EPIC.md frontmatter tags={remote_tags} "
                    f"!= registry tags={registry_tags}"
                )
            else:
                ok("frontmatter id and tags match registry")

# ── Coverage: remote epics missing from registry ───────────────────

print("\n── Coverage")
for remote_epic in sorted(remote_dirs):
    if not (registry_dir / f"{remote_epic}.json").exists():
        error(f"remote epic '{remote_epic}' has no registry file")
    else:
        ok(f"{remote_epic} covered")

# ── Summary ────────────────────────────────────────────────────────

print(f"\n{'━' * 45}")
print(f"Checked {checked} registry entries")
if errors > 0:
    print(f"{RED}{errors} error(s), {warnings} warning(s){RESET}")
    sys.exit(1)
elif warnings > 0:
    print(f"{YELLOW}0 errors, {warnings} warning(s){RESET}")
else:
    print(f"{GREEN}All checks passed{RESET}")
PYTHON
