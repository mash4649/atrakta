#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/../.." && pwd)"
DRAFT_FILE="$ROOT_DIR/.github/issue_drafts/parity_extension_brownfield.json"
EPIC_MAP_FILE="$ROOT_DIR/.github/issue_drafts/parity_extension_brownfield_epic_map.json"
OUTPUT_DIR="$ROOT_DIR/.github/issue_drafts/out"

REPO=""
DRY_RUN=0
START_ID=1
LIMIT=0
SKIP_LABEL_SETUP=0
SKIP_MILESTONE_SETUP=0

usage() {
  cat <<USAGE
Usage:
  $(basename "$0") --repo <owner/repo> [options]

Options:
  --repo <owner/repo>         Target repository (required unless --dry-run)
  --draft <path>              Draft JSON path
  --epic-map <path>           Epic/Story mapping JSON path
  --start-id <n>              Start from draft issue id (default: 1)
  --limit <n>                 Create only n issues (default: all)
  --dry-run                   Render bodies only, no GitHub API call
  --skip-label-setup          Do not create missing labels
  --skip-milestone-setup      Do not create missing milestones
  -h, --help                  Show this help

Examples:
  $(basename "$0") --repo mash4649/atrakta --dry-run
  $(basename "$0") --repo mash4649/atrakta
  $(basename "$0") --repo mash4649/atrakta --start-id 9 --limit 5
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
    --draft)
      DRAFT_FILE="${2:-}"
      shift 2
      ;;
    --epic-map)
      EPIC_MAP_FILE="${2:-}"
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
    --dry-run)
      DRY_RUN=1
      shift
      ;;
    --skip-label-setup)
      SKIP_LABEL_SETUP=1
      shift
      ;;
    --skip-milestone-setup)
      SKIP_MILESTONE_SETUP=1
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

require_cmd jq
[[ -f "$DRAFT_FILE" ]] || { echo "error: draft file not found: $DRAFT_FILE" >&2; exit 1; }
[[ -f "$EPIC_MAP_FILE" ]] || { echo "error: epic map file not found: $EPIC_MAP_FILE" >&2; exit 1; }

if [[ "$DRY_RUN" -eq 0 ]]; then
  require_cmd gh
  [[ -n "$REPO" ]] || { echo "error: --repo is required when not using --dry-run" >&2; exit 1; }
  gh auth status >/dev/null
fi

mkdir -p "$OUTPUT_DIR"
TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT

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

jq -n '{}' > "$TMP_DIR/id_map.json"

create_label_if_missing() {
  local label="$1"
  if gh label create "$label" --repo "$REPO" --color BFDADC --description "Imported by create_parity_issue_pack.sh" >/dev/null 2>"$TMP_DIR/label-create.err"; then
    return 0
  fi
  if grep -qi "already exists" "$TMP_DIR/label-create.err"; then
    return 0
  fi
  cat "$TMP_DIR/label-create.err" >&2
  return 1
}

ensure_milestone() {
  local title="$1"
  local desc="$2"
  local exists
  exists="$(gh api "repos/$REPO/milestones?state=all&per_page=100" | jq -r --arg t "$title" '.[] | select(.title==$t) | .number' | head -n1 || true)"
  if [[ -n "$exists" ]]; then
    return 0
  fi
  gh api --method POST "repos/$REPO/milestones" -f title="$title" -f description="$desc" >/dev/null
}

render_issue_body() {
  local issue_id="$1"
  local out_file="$2"

  local summary background
  summary="$(jq -r --argjson id "$issue_id" '.issues[] | select(.id==$id) | .summary' "$DRAFT_FILE")"
  background="$(jq -r --argjson id "$issue_id" '.issues[] | select(.id==$id) | .background' "$DRAFT_FILE")"

  local epic_id epic_title story_id story_title
  epic_id="$(jq -r --argjson id "$issue_id" '.issue_story_links[] | select(.issue==$id) | .epic' "$EPIC_MAP_FILE" | head -n1)"
  story_id="$(jq -r --argjson id "$issue_id" '.issue_story_links[] | select(.issue==$id) | .story' "$EPIC_MAP_FILE" | head -n1)"
  epic_title="$(jq -r --arg e "$epic_id" '.epics[] | select(.id==$e) | .title' "$EPIC_MAP_FILE" | head -n1)"
  story_title="$(jq -r --arg e "$epic_id" --arg s "$story_id" '.epics[] | select(.id==$e) | .stories[] | select(.id==$s) | .title' "$EPIC_MAP_FILE" | head -n1)"

  {
    echo "## Summary"
    echo
    echo "$summary"
    echo
    echo "## Background"
    echo
    echo "$background"
    echo
    if [[ -n "$epic_id" || -n "$story_id" ]]; then
      echo "## Epic / Story Alignment"
      echo
      [[ -n "$epic_id" ]] && echo "- Epic: \`$epic_id\` $epic_title"
      [[ -n "$story_id" ]] && echo "- Story: \`$story_id\` $story_title"
      echo
    fi

    echo "## Scope"
    echo
    jq -r --argjson id "$issue_id" '.issues[] | select(.id==$id) | .scope[]? | "- `" + . + "`"' "$DRAFT_FILE"
    echo

    echo "## Tasks"
    echo
    jq -r --argjson id "$issue_id" '.issues[] | select(.id==$id) | .tasks[]? | "- [ ] " + .' "$DRAFT_FILE"
    jq -r --argjson id "$issue_id" '.issue_overrides[]? | select(.issue==$id) | .extra_tasks[]? | "- [ ] " + .' "$EPIC_MAP_FILE"
    echo

    echo "## Acceptance Criteria"
    echo
    jq -r --argjson id "$issue_id" '.issues[] | select(.id==$id) | .acceptance[]? | "- [ ] " + .' "$DRAFT_FILE"
    jq -r --argjson id "$issue_id" '.issue_overrides[]? | select(.issue==$id) | .extra_acceptance[]? | "- [ ] " + .' "$EPIC_MAP_FILE"
    echo

    echo "## Depends on (Draft IDs)"
    echo
    if jq -e --argjson id "$issue_id" '.issues[] | select(.id==$id) | (.depends_on|length)>0' "$DRAFT_FILE" >/dev/null; then
      jq -r --argjson id "$issue_id" '.issues[] | select(.id==$id) | .depends_on[] | "- #" + (tostring)' "$DRAFT_FILE"
    else
      echo "- none"
    fi
    echo

    echo "## Tracking"
    echo
    echo "- Draft Issue ID: \`$issue_id\`"
  } > "$out_file"
}

if [[ "$DRY_RUN" -eq 0 ]]; then
  if [[ "$SKIP_LABEL_SETUP" -eq 0 ]]; then
    ALL_LABELS=($(jq -r '.issues[].labels[]' "$DRAFT_FILE" | sort -u))
    for label in "${ALL_LABELS[@]}"; do
      create_label_if_missing "$label"
    done
  fi

  if [[ "$SKIP_MILESTONE_SETUP" -eq 0 ]]; then
    while IFS=$'\t' read -r title desc; do
      [[ -n "$title" ]] || continue
      ensure_milestone "$title" "$desc"
    done < <(jq -r '.milestones[] | [.title, .description] | @tsv' "$DRAFT_FILE")
  fi
fi

for issue_id in "${ISSUE_IDS[@]}"; do
  title="$(jq -r --argjson id "$issue_id" '.issues[] | select(.id==$id) | .title' "$DRAFT_FILE")"
  milestone_key="$(jq -r --argjson id "$issue_id" '.issues[] | select(.id==$id) | .milestone' "$DRAFT_FILE")"
  milestone_title="$(jq -r --arg key "$milestone_key" '.milestones[] | select(.id==$key) | .title' "$DRAFT_FILE")"

  labels=($(jq -r --argjson id "$issue_id" '.issues[] | select(.id==$id) | .labels[]' "$DRAFT_FILE"))

  body_file="$TMP_DIR/issue-${issue_id}.md"
  render_issue_body "$issue_id" "$body_file"

  if [[ "$DRY_RUN" -eq 1 ]]; then
    echo "----- [DRY-RUN] Issue $issue_id: $title"
    sed -n '1,120p' "$body_file"
    echo
    continue
  fi

  cmd=(gh issue create --repo "$REPO" --title "$title" --body-file "$body_file")
  for l in "${labels[@]}"; do
    cmd+=(--label "$l")
  done
  if [[ -n "$milestone_title" && "$milestone_title" != "null" ]]; then
    cmd+=(--milestone "$milestone_title")
  fi

  url="$("${cmd[@]}")"
  number="${url##*/}"

  jq --argjson id "$issue_id" --argjson number "$number" '. + {($id|tostring): $number}' "$TMP_DIR/id_map.json" > "$TMP_DIR/id_map.next.json"
  mv "$TMP_DIR/id_map.next.json" "$TMP_DIR/id_map.json"

  echo "created: draft#$issue_id -> #$number ($url)"
done

if [[ "$DRY_RUN" -eq 1 ]]; then
  echo "dry-run complete"
  exit 0
fi

# Add dependency comments after all issues are created.
for issue_id in "${ISSUE_IDS[@]}"; do
  created_number="$(jq -r --arg id "$issue_id" '.[$id]' "$TMP_DIR/id_map.json")"
  [[ "$created_number" != "null" ]] || continue

  deps=($(jq -r --argjson id "$issue_id" '.issues[] | select(.id==$id) | .depends_on[]?' "$DRAFT_FILE"))
  if [[ ${#deps[@]} -eq 0 ]]; then
    continue
  fi

  dep_refs=()
  for dep in "${deps[@]}"; do
    mapped="$(jq -r --arg id "$dep" '.[$id]' "$TMP_DIR/id_map.json")"
    if [[ -n "$mapped" && "$mapped" != "null" ]]; then
      dep_refs+=("#$mapped")
    else
      dep_refs+=("draft#$dep")
    fi
  done

  dep_line="${dep_refs[*]}"
  gh issue comment "$created_number" --repo "$REPO" --body "Depends on: $dep_line"
  echo "linked deps: #$created_number <- $dep_line"
done

timestamp="$(date +%Y%m%d-%H%M%S)"
out_file="$OUTPUT_DIR/issue-map-$timestamp.json"
cp "$TMP_DIR/id_map.json" "$out_file"

echo "done"
echo "issue map: $out_file"
