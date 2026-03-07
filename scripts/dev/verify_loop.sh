#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

mkdir -p .tmp/go-build .tmp/go-mod
export GOCACHE="${GOCACHE:-$ROOT_DIR/.tmp/go-build}"
export GOMODCACHE="${GOMODCACHE:-$ROOT_DIR/.tmp/go-mod}"

echo "[verify-loop] go test ./..."
if go test ./...; then
  echo "[verify-loop] PASS on first attempt"
  exit 0
fi

echo "[verify-loop] first attempt failed, running doctor recovery"
go run ./cmd/atrakta doctor >/tmp/atrakta-doctor.out 2>&1 || true
cat /tmp/atrakta-doctor.out

echo "[verify-loop] retry go test ./..."
go test ./...
echo "[verify-loop] PASS after recovery retry"
