#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

mkdir -p .tmp/go-build .tmp/go-mod .tmp/soak/logs
export GOCACHE="${GOCACHE:-$ROOT_DIR/.tmp/go-build}"
export GOMODCACHE="${GOMODCACHE:-$ROOT_DIR/.tmp/go-mod}"

SOAK_HOURS="${SOAK_HOURS:-0}"
SOAK_MINUTES="${SOAK_MINUTES:-0}"
SOAK_ITERATIONS="${SOAK_ITERATIONS:-0}"
SOAK_SLEEP_SEC="${SOAK_SLEEP_SEC:-2}"
SOAK_MAX_FAILURES="${SOAK_MAX_FAILURES:-0}"
SOAK_WORKSPACE="${SOAK_WORKSPACE:-$ROOT_DIR/.tmp/soak/workspace}"
SOAK_LOG_DIR="${SOAK_LOG_DIR:-$ROOT_DIR/.tmp/soak/logs}"
SOAK_BIN="${SOAK_BIN:-$ROOT_DIR/.tmp/soak/atrakta-soak}"
SOAK_INTERFACES="${SOAK_INTERFACES:-cursor}"
SOAK_FEATURE_ID="${SOAK_FEATURE_ID:-soak}"

usage() {
  cat <<'EOF'
usage: ./scripts/soak.sh [--hours N] [--minutes N] [--iterations N] [--sleep-sec N] [--max-failures N]

Runs repeated `atrakta start/doctor/migrate check` cycles for long-duration soak validation.
At least one of --hours / --minutes / --iterations is required.
EOF
}

while [ $# -gt 0 ]; do
  case "$1" in
    --hours)
      SOAK_HOURS="${2:-0}"
      shift 2
      ;;
    --minutes)
      SOAK_MINUTES="${2:-0}"
      shift 2
      ;;
    --iterations)
      SOAK_ITERATIONS="${2:-0}"
      shift 2
      ;;
    --sleep-sec)
      SOAK_SLEEP_SEC="${2:-2}"
      shift 2
      ;;
    --max-failures)
      SOAK_MAX_FAILURES="${2:-0}"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "unknown argument: $1"
      usage
      exit 2
      ;;
  esac
done

duration_sec=$((SOAK_HOURS * 3600 + SOAK_MINUTES * 60))
if [ "$duration_sec" -le 0 ] && [ "$SOAK_ITERATIONS" -le 0 ]; then
  echo "error: set --hours, --minutes, or --iterations"
  exit 2
fi

now_ms() {
  if command -v perl >/dev/null 2>&1; then
    perl -MTime::HiRes=time -e 'printf("%.0f\n", time()*1000)'
  else
    echo "$(( $(date +%s) * 1000 ))"
  fi
}

percentile_value() {
  local file="$1"
  local p="$2"
  local n rank
  n="$(awk 'NF{c++} END{print c+0}' "$file")"
  if [ "$n" -le 0 ]; then
    echo "0"
    return
  fi
  rank="$(awk -v n="$n" -v p="$p" 'BEGIN{
    r = int((n * p + 99) / 100)
    if (r < 1) r = 1
    if (r > n) r = n
    print r
  }')"
  sort -n "$file" | sed -n "${rank}p"
}

echo "soak: building atrakta binary -> $SOAK_BIN"
go build -o "$SOAK_BIN" ./cmd/atrakta

mkdir -p "$SOAK_WORKSPACE" "$SOAK_LOG_DIR"
if [ ! -f "$SOAK_WORKSPACE/AGENTS.md" ]; then
  cat > "$SOAK_WORKSPACE/AGENTS.md" <<'EOF'
# Atrakta Soak Workspace
EOF
fi

durations_file="$SOAK_LOG_DIR/durations_ms.txt"
: > "$durations_file"

started_at="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
start_epoch="$(date +%s)"
end_epoch=0
if [ "$duration_sec" -gt 0 ]; then
  end_epoch=$((start_epoch + duration_sec))
fi

iterations=0
failures=0

echo "soak: started_at=$started_at workspace=$SOAK_WORKSPACE interfaces=$SOAK_INTERFACES feature_id=$SOAK_FEATURE_ID"
while true; do
  now_epoch="$(date +%s)"
  if [ "$end_epoch" -gt 0 ] && [ "$now_epoch" -ge "$end_epoch" ]; then
    break
  fi
  if [ "$SOAK_ITERATIONS" -gt 0 ] && [ "$iterations" -ge "$SOAK_ITERATIONS" ]; then
    break
  fi
  iterations=$((iterations + 1))
  iter_log="$SOAK_LOG_DIR/iter-$iterations.log"
  t0="$(now_ms)"
  status=0
  (
    cd "$SOAK_WORKSPACE"
    "$SOAK_BIN" start --interfaces "$SOAK_INTERFACES" --feature-id "$SOAK_FEATURE_ID"
    "$SOAK_BIN" doctor --sync-proposal
    "$SOAK_BIN" migrate check
  ) >"$iter_log" 2>&1 || status=$?
  t1="$(now_ms)"
  elapsed_ms=$((t1 - t0))
  echo "$elapsed_ms" >> "$durations_file"
  if [ "$status" -ne 0 ]; then
    failures=$((failures + 1))
    echo "soak: iteration=$iterations failed status=$status elapsed_ms=$elapsed_ms log=$iter_log"
  fi
  if [ "$failures" -gt "$SOAK_MAX_FAILURES" ]; then
    echo "soak: failure budget exceeded (failures=$failures, max=$SOAK_MAX_FAILURES)"
    break
  fi
  sleep "$SOAK_SLEEP_SEC"
done

p50_ms="$(percentile_value "$durations_file" 50)"
p95_ms="$(percentile_value "$durations_file" 95)"
finished_at="$(date -u +%Y-%m-%dT%H:%M:%SZ)"

echo "soak: finished_at=$finished_at iterations=$iterations failures=$failures p50_ms=$p50_ms p95_ms=$p95_ms"

if [ "$failures" -gt "$SOAK_MAX_FAILURES" ]; then
  exit 1
fi

