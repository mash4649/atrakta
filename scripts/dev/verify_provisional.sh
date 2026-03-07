#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT_DIR"

mkdir -p .tmp/go-build .tmp/go-mod .tmp/dist
export GOCACHE="$ROOT_DIR/.tmp/go-build"
export GOMODCACHE="$ROOT_DIR/.tmp/go-mod"

THRESHOLD_NS=50000000

echo "[1/5] formatting check + base tests"
gofmt -w $(find cmd internal -name '*.go')
./scripts/dev/verify_loop.sh

echo "[2/5] fast-path benchmark"
BENCH_OUT="$(go test ./internal/wrapper -run '^$' -bench '^BenchmarkWrapperFastPath$' -benchmem -count=5)"
echo "$BENCH_OUT"

FAST_NS="$(printf '%s\n' "$BENCH_OUT" | awk '
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
  if (last == "") {
    exit 2
  }
  print last
}
')"

if [ "$FAST_NS" -gt "$THRESHOLD_NS" ]; then
  echo "fast-path benchmark failed: ${FAST_NS}ns/op > ${THRESHOLD_NS}ns/op"
  exit 1
fi

echo "fast-path benchmark passed: ${FAST_NS}ns/op <= ${THRESHOLD_NS}ns/op"

echo "[3/5] cross-build test compile verification"
for target in linux/amd64 windows/amd64; do
  goos="${target%/*}"
  goarch="${target#*/}"
  echo "  - GOOS=$goos GOARCH=$goarch"
  GOOS="$goos" GOARCH="$goarch" CGO_ENABLED=0 go test ./... -run '^$' -exec /usr/bin/true
 done

echo "[4/5] cross-build binary generation"
for target in linux/amd64 windows/amd64; do
  goos="${target%/*}"
  goarch="${target#*/}"
  ext=""
  if [ "$goos" = "windows" ]; then
    ext=".exe"
  fi
  out=".tmp/dist/atrakta-${goos}-${goarch}${ext}"
  GOOS="$goos" GOARCH="$goarch" CGO_ENABLED=0 go build -o "$out" ./cmd/atrakta
  if [ ! -s "$out" ]; then
    echo "binary verification failed: $out not generated or empty"
    exit 1
  fi
  echo "  - built $out"
done

echo "[5/5] artifact summary"
ls -lh .tmp/dist/atrakta-*

echo "provisional verification passed"
