#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT_DIR"

VERSION="$(tr -d '\n' < VERSION)"
if [ -z "$VERSION" ]; then
  echo "VERSION is empty"
  exit 1
fi

mkdir -p .tmp/go-build .tmp/go-mod
export GOCACHE="${GOCACHE:-$ROOT_DIR/.tmp/go-build}"
export GOMODCACHE="${GOMODCACHE:-$ROOT_DIR/.tmp/go-mod}"

OUT_DIR="${OUT_DIR:-$ROOT_DIR/.tmp/release/v${VERSION}}"
DIST_DIR="$OUT_DIR/dist"
PKG_DIR="$OUT_DIR/packages"
mkdir -p "$DIST_DIR" "$PKG_DIR"

TARGETS=(
  "darwin/arm64"
  "darwin/amd64"
  "linux/amd64"
  "linux/arm64"
  "windows/amd64"
)

for target in "${TARGETS[@]}"; do
  goos="${target%/*}"
  goarch="${target#*/}"
  ext=""
  if [ "$goos" = "windows" ]; then
    ext=".exe"
  fi
  build_bin_name="atrakta-${goos}-${goarch}${ext}"
  archive_bin_name="atrakta${ext}"
  out_bin="$DIST_DIR/$build_bin_name"
  echo "build: $goos/$goarch -> $out_bin"
  GOOS="$goos" GOARCH="$goarch" CGO_ENABLED=0 go build -trimpath -o "$out_bin" ./cmd/atrakta

  pkg_base="atrakta_${VERSION}_${goos}_${goarch}"
  if [ "$goos" = "windows" ]; then
    pkg_file="$PKG_DIR/${pkg_base}.zip"
    rm -f "$pkg_file"
    cp "$out_bin" "$DIST_DIR/$archive_bin_name"
    (
      cd "$DIST_DIR"
      zip -q -9 "$pkg_file" "$archive_bin_name"
    )
    rm -f "$DIST_DIR/$archive_bin_name"
  else
    pkg_file="$PKG_DIR/${pkg_base}.tar.gz"
    cp "$out_bin" "$DIST_DIR/$archive_bin_name"
    tar -C "$DIST_DIR" -czf "$pkg_file" "$archive_bin_name"
    rm -f "$DIST_DIR/$archive_bin_name"
  fi
done

CHECKSUM_FILE="$OUT_DIR/checksums.txt"
(
  cd "$PKG_DIR"
  shasum -a 256 ./* | sort > "$CHECKSUM_FILE"
)

echo "release artifacts ready:"
echo "  packages: $PKG_DIR"
echo "  checksums: $CHECKSUM_FILE"
ls -lh "$PKG_DIR"
