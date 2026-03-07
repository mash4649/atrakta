#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT_DIR"

mkdir -p .tmp/go-build .tmp/go-mod
export GOCACHE="$ROOT_DIR/.tmp/go-build"
export GOMODCACHE="$ROOT_DIR/.tmp/go-mod"

SAMPLE_COUNT="${PERF_SAMPLE_COUNT:-20}"
APPLY_OPS300_P95_NS="${APPLY_OPS300_P95_NS:-25000000}"
APPLY_OPS300_P99_NS="${APPLY_OPS300_P99_NS:-35000000}"
PLAN_NOOP1000_P95_NS="${PLAN_NOOP1000_P95_NS:-25000000}"
PLAN_NOOP1000_P99_NS="${PLAN_NOOP1000_P99_NS:-40000000}"
WRAPPER_FASTPATH_P95_NS="${WRAPPER_FASTPATH_P95_NS:-30000000}"
WRAPPER_FASTPATH_P99_NS="${WRAPPER_FASTPATH_P99_NS:-50000000}"
START_STEADY_P95_NS="${START_STEADY_P95_NS:-1200000000}"
START_STEADY_P99_NS="${START_STEADY_P99_NS:-1800000000}"

extract_ns_samples() {
  local prefix="$1"
  awk -v prefix="$prefix" '
index($1, prefix) == 1 {
  for (i = 1; i <= NF; i++) {
    if ($i == "ns/op") {
      v = $(i-1)
      gsub(/,/, "", v)
      print v
    }
  }
}
'
}

percentile_value() {
  local samples="$1"
  local p="$2"
  local n rank
  n="$(printf '%s\n' "$samples" | awk 'NF{c++} END{print c+0}')"
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
  printf '%s\n' "$samples" | sort -n | sed -n "${rank}p"
}

run_gate() {
  local label="$1"
  local pkg="$2"
  local bench_regex="$3"
  local sample_prefix="$4"
  local p95_limit="$5"
  local p99_limit="$6"

  local out samples sample_n p95 p99
  # Warm-up run to avoid counting one-off cold-start noise in percentile gates.
  go test "$pkg" -run '^$' -bench "$bench_regex" -benchmem -count=1 >/dev/null
  out="$(go test "$pkg" -run '^$' -bench "$bench_regex" -benchmem -count="$SAMPLE_COUNT")"
  printf '%s\n' "$out"
  samples="$(printf '%s\n' "$out" | extract_ns_samples "$sample_prefix")"
  sample_n="$(printf '%s\n' "$samples" | awk 'NF{c++} END{print c+0}')"
  if [ "$sample_n" -lt 3 ]; then
    echo "FAIL: ${label} has insufficient samples (${sample_n})"
    exit 1
  fi
  p95="$(percentile_value "$samples" 95)"
  p99="$(percentile_value "$samples" 99)"
  echo "gate ${label}: p95=${p95}ns p99=${p99}ns (limits p95<=${p95_limit}, p99<=${p99_limit})"
  if [ "$p95" -gt "$p95_limit" ] || [ "$p99" -gt "$p99_limit" ]; then
    echo "FAIL: ${label} regression gate exceeded"
    exit 1
  fi
}

echo "slo gate: samples=${SAMPLE_COUNT}"
run_gate \
  "apply_ops_300" \
  "./internal/apply" \
  '^BenchmarkApplyScaling/ops_300$' \
  'BenchmarkApplyScaling/ops_300' \
  "$APPLY_OPS300_P95_NS" \
  "$APPLY_OPS300_P99_NS"

run_gate \
  "plan_noop_managed_1000" \
  "./internal/plan" \
  '^BenchmarkBuildNoopManagedScaling/managed_1000$' \
  'BenchmarkBuildNoopManagedScaling/managed_1000' \
  "$PLAN_NOOP1000_P95_NS" \
  "$PLAN_NOOP1000_P99_NS"

run_gate \
  "wrapper_fastpath" \
  "./internal/wrapper" \
  '^BenchmarkWrapperFastPath$' \
  'BenchmarkWrapperFastPath' \
  "$WRAPPER_FASTPATH_P95_NS" \
  "$WRAPPER_FASTPATH_P99_NS"

run_gate \
  "start_steady_state" \
  "./internal/core" \
  '^BenchmarkStartSteadyState$' \
  'BenchmarkStartSteadyState' \
  "$START_STEADY_P95_NS" \
  "$START_STEADY_P99_NS"

echo "gate token_budget"
go test ./internal/core -run '^TestSLORepoMapTokenBudgetRespected$' -count=1

echo "gate fastpath_hit_rate"
go test ./internal/wrapper -run '^TestSLOWrapperFastPathHitRate$' -count=1

echo "slo gate passed"
