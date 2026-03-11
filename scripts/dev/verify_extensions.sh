#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT_DIR"

mkdir -p .tmp/go-build .tmp/verify
LOG_FILE=".tmp/verify/verify_extensions.log"
START_TS="$(date +%s)"
CMD="GOCACHE=$(pwd)/.tmp/go-build go test ./internal/contract ./internal/projection"

set +e
bash -lc "$CMD" >"$LOG_FILE" 2>&1
EXIT_CODE=$?
set -e

END_TS="$(date +%s)"
DURATION="$((END_TS - START_TS))"
STATUS="pass"
if [[ "$EXIT_CODE" -ne 0 ]]; then
  STATUS="fail"
fi

cat <<JSON
{
  "verify": "extensions",
  "status": "$STATUS",
  "exit_code": $EXIT_CODE,
  "duration_sec": $DURATION,
  "command": "$CMD",
  "log_file": "$LOG_FILE"
}
JSON

exit "$EXIT_CODE"
