#!/usr/bin/env bash
set -euo pipefail

OWNER=""
PROJECT_NUMBER=""
DRY_RUN=0

usage() {
  cat <<USAGE
Usage:
  $(basename "$0") --owner <owner> --project-number <n> [options]

Options:
  --owner <owner>           Project owner login (required)
  --project-number <n>      Project number (required)
  --dry-run                 Print actions only
  -h, --help                Show this help

Behavior:
  - Ensures a SINGLE_SELECT field named "Priority" exists with options P0,P1,P2.
  - Syncs each project item's priority label (priority:P0/P1/P2) to the Priority field.
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
    --owner)
      OWNER="${2:-}"
      shift 2
      ;;
    --project-number)
      PROJECT_NUMBER="${2:-}"
      shift 2
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

[[ -n "$OWNER" ]] || { echo "error: --owner is required" >&2; exit 1; }
[[ -n "$PROJECT_NUMBER" ]] || { echo "error: --project-number is required" >&2; exit 1; }

if [[ "$DRY_RUN" -eq 0 ]]; then
  gh auth status >/dev/null
fi

project_json="$(gh project view "$PROJECT_NUMBER" --owner "$OWNER" --format json)"
project_id="$(echo "$project_json" | jq -r '.id')"

fields_json="$(gh project field-list "$PROJECT_NUMBER" --owner "$OWNER" --limit 200 --format json)"
priority_field_id="$(echo "$fields_json" | jq -r '.fields[] | select(.name=="Priority") | .id' | head -n1 || true)"

if [[ -z "$priority_field_id" ]]; then
  if [[ "$DRY_RUN" -eq 1 ]]; then
    echo "[dry-run] would create Priority field with options P0,P1,P2"
  else
    gh project field-create "$PROJECT_NUMBER" --owner "$OWNER" --name "Priority" --data-type "SINGLE_SELECT" --single-select-options "P0,P1,P2" >/dev/null
    echo "created field: Priority"
  fi
  fields_json="$(gh project field-list "$PROJECT_NUMBER" --owner "$OWNER" --limit 200 --format json)"
  priority_field_id="$(echo "$fields_json" | jq -r '.fields[] | select(.name=="Priority") | .id' | head -n1 || true)"
fi

p0_option_id="$(echo "$fields_json" | jq -r '.fields[] | select(.name=="Priority") | .options[] | select(.name=="P0") | .id' | head -n1 || true)"
p1_option_id="$(echo "$fields_json" | jq -r '.fields[] | select(.name=="Priority") | .options[] | select(.name=="P1") | .id' | head -n1 || true)"
p2_option_id="$(echo "$fields_json" | jq -r '.fields[] | select(.name=="Priority") | .options[] | select(.name=="P2") | .id' | head -n1 || true)"

[[ -n "$priority_field_id" ]] || { echo "error: Priority field id not found" >&2; exit 1; }
[[ -n "$p0_option_id" ]] || { echo "error: Priority option P0 not found" >&2; exit 1; }
[[ -n "$p1_option_id" ]] || { echo "error: Priority option P1 not found" >&2; exit 1; }
[[ -n "$p2_option_id" ]] || { echo "error: Priority option P2 not found" >&2; exit 1; }

items_json="$(gh project item-list "$PROJECT_NUMBER" --owner "$OWNER" --limit 500 --format json)"

updates=0
skipped=0

while IFS=$'\t' read -r item_id issue_number labels_csv; do
  [[ -n "$item_id" ]] || continue

  option_id=""
  case "$labels_csv" in
    *priority:P0*) option_id="$p0_option_id" ;;
    *priority:P1*) option_id="$p1_option_id" ;;
    *priority:P2*) option_id="$p2_option_id" ;;
    *) option_id="" ;;
  esac

  if [[ -z "$option_id" ]]; then
    echo "skip: issue #$issue_number has no priority label"
    skipped=$((skipped + 1))
    continue
  fi

  if [[ "$DRY_RUN" -eq 1 ]]; then
    echo "[dry-run] set Priority for issue #$issue_number (item=$item_id)"
  else
    gh project item-edit --id "$item_id" --project-id "$project_id" --field-id "$priority_field_id" --single-select-option-id "$option_id" >/dev/null
    echo "updated: issue #$issue_number"
  fi

  updates=$((updates + 1))
done < <(echo "$items_json" | jq -r '.items[] | [.id, .content.number, (.labels|join(","))] | @tsv')

echo "done"
echo "project_number=$PROJECT_NUMBER updates=$updates skipped=$skipped"
