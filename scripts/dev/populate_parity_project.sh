#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/../.." && pwd)"
DRAFT_FILE="$ROOT_DIR/.github/issue_drafts/parity_extension_brownfield.json"
ISSUE_MAP_FILE=""

OWNER=""
REPO=""
PROJECT_TITLE="Atrakta Parity / Extension / Brownfield Backlog"
PROJECT_NUMBER=""
START_ID=1
LIMIT=0
DRY_RUN=0
SKIP_LINK_REPO=0

usage() {
  cat <<USAGE
Usage:
  $(basename "$0") --repo <owner/repo> [options]

Options:
  --repo <owner/repo>         Target repository for issue URLs (required)
  --owner <owner>             Project owner (default: derived from --repo)
  --project-title <title>     Project title (used when --project-number is not provided)
  --project-number <n>        Existing project number to use
  --draft <path>              Draft JSON path
  --issue-map <path>          Optional draft-id -> issue-number JSON map
  --start-id <n>              Start from draft issue id (default: 1)
  --limit <n>                 Add only n issues (default: all)
  --skip-link-repo            Skip linking project to repository
  --dry-run                   Print actions only
  -h, --help                  Show this help

Examples:
  $(basename "$0") --repo mash4649/atrakta
  $(basename "$0") --repo mash4649/atrakta --project-number 1
  $(basename "$0") --repo mash4649/atrakta --issue-map .github/issue_drafts/out/issue-map-merged.json
USAGE
}

require_cmd() {
  local cmd="$1"
  if ! command -v "$cmd" >/dev/null 2>&1; then
    echo "error: required command not found: $cmd" >&2
    exit 1
  fi
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --repo)
      REPO="${2:-}"
      shift 2
      ;;
    --owner)
      OWNER="${2:-}"
      shift 2
      ;;
    --project-title)
      PROJECT_TITLE="${2:-}"
      shift 2
      ;;
    --project-number)
      PROJECT_NUMBER="${2:-}"
      shift 2
      ;;
    --draft)
      DRAFT_FILE="${2:-}"
      shift 2
      ;;
    --issue-map)
      ISSUE_MAP_FILE="${2:-}"
      shift 2
      ;;
    --start-id)
      START_ID="${2:-}"
      shift 2
      ;;
    --limit)
      LIMIT="${2:-}"
      shift 2
      ;;
    --skip-link-repo)
      SKIP_LINK_REPO=1
      shift
      ;;
    --dry-run)
      DRY_RUN=1
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "error: unknown arg: $1" >&2
      usage >&2
      exit 1
      ;;
  esac
done

require_cmd gh
require_cmd jq

[[ -n "$REPO" ]] || { echo "error: --repo is required" >&2; exit 1; }
[[ -f "$DRAFT_FILE" ]] || { echo "error: draft file not found: $DRAFT_FILE" >&2; exit 1; }

if [[ -z "$OWNER" ]]; then
  OWNER="${REPO%%/*}"
fi

if [[ -n "$ISSUE_MAP_FILE" && ! -f "$ISSUE_MAP_FILE" ]]; then
  echo "error: issue map not found: $ISSUE_MAP_FILE" >&2
  exit 1
fi

if [[ "$DRY_RUN" -eq 0 ]]; then
  gh auth status >/dev/null
fi

ISSUE_IDS=($(jq -r --argjson start "$START_ID" --argjson limit "$LIMIT" '
  [ .issues[] | select(.id >= $start) | .id ]
  | sort
  | if $limit > 0 then .[:$limit] else . end
  | .[]
' "$DRAFT_FILE"))

if [[ ${#ISSUE_IDS[@]} -eq 0 ]]; then
  echo "no issues selected (start-id=$START_ID, limit=$LIMIT)"
  exit 0
fi

if [[ -z "$PROJECT_NUMBER" ]]; then
  PROJECT_NUMBER="$(gh project list --owner "$OWNER" --limit 100 --format json | jq -r --arg t "$PROJECT_TITLE" '.projects[] | select(.title==$t) | .number' | head -n1 || true)"
  if [[ -z "$PROJECT_NUMBER" ]]; then
    if [[ "$DRY_RUN" -eq 1 ]]; then
      PROJECT_NUMBER="DRYRUN"
      echo "[dry-run] would create project: $PROJECT_TITLE (owner=$OWNER)"
    else
      PROJECT_NUMBER="$(gh project create --owner "$OWNER" --title "$PROJECT_TITLE" --format json --jq '.number')"
      echo "created project: #$PROJECT_NUMBER ($PROJECT_TITLE)"
    fi
  else
    echo "using existing project: #$PROJECT_NUMBER ($PROJECT_TITLE)"
  fi
fi

if [[ "$SKIP_LINK_REPO" -eq 0 ]]; then
  if [[ "$DRY_RUN" -eq 1 ]]; then
    echo "[dry-run] would link project #$PROJECT_NUMBER to repo $REPO"
  else
    if ! gh project link "$PROJECT_NUMBER" --owner "$OWNER" --repo "$REPO" >/tmp/project-link.err 2>&1; then
      if grep -qi "already" /tmp/project-link.err; then
        :
      else
        cat /tmp/project-link.err >&2
        rm -f /tmp/project-link.err
        exit 1
      fi
    fi
    rm -f /tmp/project-link.err
  fi
fi

TMP_FILE="$(mktemp)"
trap 'rm -f "$TMP_FILE"' EXIT

if [[ "$DRY_RUN" -eq 0 ]]; then
  gh project item-list "$PROJECT_NUMBER" --owner "$OWNER" --limit 500 --format json | jq -r '.items[].content.url // empty' > "$TMP_FILE"
else
  : > "$TMP_FILE"
fi

added=0
skipped=0

resolve_issue_number() {
  local draft_id="$1"
  if [[ -n "$ISSUE_MAP_FILE" ]]; then
    jq -r --arg id "$draft_id" '.[$id] // empty' "$ISSUE_MAP_FILE"
  else
    echo "$draft_id"
  fi
}

for draft_id in "${ISSUE_IDS[@]}"; do
  issue_number="$(resolve_issue_number "$draft_id")"
  if [[ -z "$issue_number" || "$issue_number" == "null" ]]; then
    echo "skip: draft#$draft_id has no mapped issue number"
    skipped=$((skipped + 1))
    continue
  fi

  issue_url="https://github.com/$REPO/issues/$issue_number"

  if grep -Fxq "$issue_url" "$TMP_FILE"; then
    echo "skip: already in project ($issue_url)"
    skipped=$((skipped + 1))
    continue
  fi

  if [[ "$DRY_RUN" -eq 1 ]]; then
    echo "[dry-run] add: draft#$draft_id -> $issue_url"
  else
    gh project item-add "$PROJECT_NUMBER" --owner "$OWNER" --url "$issue_url" >/dev/null
    echo "added: draft#$draft_id -> $issue_url"
    echo "$issue_url" >> "$TMP_FILE"
  fi

  added=$((added + 1))
done

echo "done"
echo "project_number=$PROJECT_NUMBER"
echo "added=$added skipped=$skipped"
