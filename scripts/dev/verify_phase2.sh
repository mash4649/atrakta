#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

mkdir -p .tmp/go-build .tmp/go-mod .tmp/dist .tmp/linux-e2e
export GOCACHE="$ROOT_DIR/.tmp/go-build"
export GOMODCACHE="$ROOT_DIR/.tmp/go-mod"

FAST_THRESHOLD_NS=50000000

echo "[1/13] format + full test"
gofmt -w $(find cmd internal -name '*.go')
./scripts/verify_loop.sh

echo "[2/13] wrapper fast-path benchmark (<50ms)"
FAST_OUT="$(go test ./internal/wrapper -run '^$' -bench '^BenchmarkWrapperFastPath$' -benchmem -count=5)"
printf '%s\n' "$FAST_OUT"
FAST_NS="$(printf '%s\n' "$FAST_OUT" | awk '
/BenchmarkWrapperFastPath/ {
  for (i=1; i<=NF; i++) {
    if ($i == "ns/op") {
      v = $(i-1)
      gsub(/,/, "", v)
      last = v
    }
  }
}
END {
  if (last == "") exit 2;
  print last;
}
')"
if [ "$FAST_NS" -gt "$FAST_THRESHOLD_NS" ]; then
  echo "FAIL: wrapper fast-path ${FAST_NS}ns/op > ${FAST_THRESHOLD_NS}ns/op"
  exit 1
fi
echo "PASS: wrapper fast-path ${FAST_NS}ns/op"

echo "[3/13] detect scaling benchmark"
go test ./internal/detect -run '^$' -bench '^BenchmarkDetectScaling$' -benchmem -count=1

echo "[4/13] apply scaling benchmark"
go test ./internal/apply -run '^$' -bench '^BenchmarkApplyScaling$' -benchmem -count=1

echo "[5/13] perf regression gate (p95/p99 fail-closed)"
./scripts/verify_perf_gate.sh

echo "[6/13] projection scaling stress"
go test ./internal/projection -run '^TestProjectionScalingLinearBound$' -count=1

echo "[7/13] H7 replay determinism migration test"
go test ./internal/migrate -run '^TestH7ReplayDeterminismWithSchemaVersion2AdditiveFields$' -count=1

echo "[8/13] phase2 autonomy safety tests"
go test ./internal/events -run '^TestAppendConcurrentChainSafe$' -count=1
go test ./internal/subworker -run '^TestMergePhaseAUsesProposals$|^TestRollbackToSingleWriterOnProposalConflict$|^TestBuildSingleWriterQueueOrdersByPathThenWorker$|^TestValidateSingleWriterQueueDetectsNonDeterministicOrder$' -count=1
go test ./internal/gitauto -run '^TestEnsureSetupAutoInitializesWhenSafe$' -count=1
go test ./internal/apply -run '^TestParallelApplyAutoForNonDestructiveOps$|^TestParallelApplyFallsBackWhenDestructiveIncluded$' -count=1
go test ./internal/core -run '^TestSubworkerBranchPlanCanBeEnabledInLimitedMode$' -count=1

echo "[9/13] cross-build test compile (linux/windows)"
for target in linux/amd64 windows/amd64; do
  goos="${target%/*}"
  goarch="${target#*/}"
  echo "  - GOOS=$goos GOARCH=$goarch"
  GOOS="$goos" GOARCH="$goarch" CGO_ENABLED=0 go test ./... -run '^$' -exec /usr/bin/true
 done

echo "[10/13] cross-build binary generation"
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o .tmp/dist/atrakta-linux-amd64 ./cmd/atrakta
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o .tmp/dist/atrakta-windows-amd64.exe ./cmd/atrakta
ls -lh .tmp/dist/atrakta-linux-amd64 .tmp/dist/atrakta-windows-amd64.exe

echo "[11/13] linux runtime execution test via docker"
cat > .tmp/linux-e2e/AGENTS.md <<'EOT'
# Linux E2E AGENTS
sync.prefer_interfaces: cursor
EOT

# Execute linux binary in a Linux container against a temp workspace.
docker run --rm \
  -v "$ROOT_DIR:/workspace" \
  -w /workspace/.tmp/linux-e2e \
  caddy:alpine \
  /bin/sh -lc '/workspace/.tmp/dist/atrakta-linux-amd64 start --interfaces cursor --feature-id feat-linux >/tmp/start.out 2>&1 && /workspace/.tmp/dist/atrakta-linux-amd64 doctor --sync-proposal >/tmp/doctor.out 2>&1 && /workspace/.tmp/dist/atrakta-linux-amd64 migrate check >/tmp/migrate.out 2>&1 && cat /tmp/start.out /tmp/doctor.out /tmp/migrate.out'

echo "[12/13] windows runtime execution test"
echo "SKIP: Windows runtime execution is not available on this Linux Docker daemon (OSType=linux)."

echo "[13/13] completed"
echo "phase2 verification completed (with Windows runtime test skipped by environment constraint)"
