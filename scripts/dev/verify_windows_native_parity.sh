#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT_DIR"

mkdir -p .tmp/go-build .tmp/verify
LOG_FILE=".tmp/verify/verify_windows_native_parity.log"
START_TS="$(date +%s)"

RUNNER_OS_VAL="${RUNNER_OS:-}"
GOOS_VAL="$(go env GOOS)"
WINDOWS_NATIVE=0
if [[ "$RUNNER_OS_VAL" == "Windows" || "$GOOS_VAL" == "windows" ]]; then
  WINDOWS_NATIVE=1
fi

if [[ "$WINDOWS_NATIVE" -ne 1 ]]; then
  if [[ "${ATRAKTA_ALLOW_WINDOWS_PARITY_SKIP:-0}" == "1" ]]; then
    END_TS="$(date +%s)"
    DURATION="$((END_TS - START_TS))"
    cat <<JSON
{
  "verify": "windows_native_parity",
  "status": "skipped",
  "exit_code": 0,
  "duration_sec": $DURATION,
  "command": "go test ./internal/core -run '^TestWindowsNativeParity$' -count=1",
  "log_file": "$LOG_FILE",
  "skip_reason": "non-windows environment"
}
JSON
    exit 0
  fi
  echo "windows native parity gate must run on windows-native runner (set ATRACTA_ALLOW_WINDOWS_PARITY_SKIP=1 to skip explicitly)." >&2
  exit 2
fi

CMD="GOCACHE=$(pwd)/.tmp/go-build go test ./internal/core -run '^TestWindowsNativeParity$' -count=1"
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
  "verify": "windows_native_parity",
  "status": "$STATUS",
  "exit_code": $EXIT_CODE,
  "duration_sec": $DURATION,
  "command": "$CMD",
  "log_file": "$LOG_FILE"
}
JSON

exit "$EXIT_CODE"
