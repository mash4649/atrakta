#!/usr/bin/env bash
set -euo pipefail

REPO="${ATRAKTA_REPO:-afwm/Atrakta}"
VERSION="${ATRAKTA_VERSION:-latest}"
INSTALL_DIR="${ATRAKTA_INSTALL_DIR:-$HOME/.local/bin}"
INSTALL_NAME="atrakta"

need_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "missing required command: $1" >&2
    exit 1
  fi
}

need_cmd curl
need_cmd tar

TMP_DIR="$(mktemp -d)"
cleanup() {
  rm -rf "$TMP_DIR"
}
trap cleanup EXIT

uname_s="$(uname -s | tr '[:upper:]' '[:lower:]')"
case "$uname_s" in
  darwin) os="darwin" ;;
  linux) os="linux" ;;
  *)
    echo "unsupported OS: $(uname -s). Use manual install for this platform." >&2
    exit 1
    ;;
esac

uname_m="$(uname -m)"
case "$uname_m" in
  x86_64|amd64) arch="amd64" ;;
  arm64|aarch64) arch="arm64" ;;
  *)
    echo "unsupported architecture: $uname_m" >&2
    exit 1
    ;;
esac

release_api="https://api.github.com/repos/${REPO}/releases/latest"
if [ "$VERSION" != "latest" ]; then
  release_api="https://api.github.com/repos/${REPO}/releases/tags/${VERSION}"
fi

echo "resolving release metadata from ${REPO} (${VERSION})..."
release_json="$TMP_DIR/release.json"
curl -fsSL "$release_api" -o "$release_json"

asset_url="$(sed -n 's/.*"browser_download_url":[[:space:]]*"\([^"]*\)".*/\1/p' "$release_json" \
  | grep -E "/atrakta_[^/]+_${os}_${arch}\\.tar\\.gz$" \
  | head -n 1 || true)"

checksums_url="$(sed -n 's/.*"browser_download_url":[[:space:]]*"\([^"]*\)".*/\1/p' "$release_json" \
  | grep -E '/checksums\.txt$' \
  | head -n 1 || true)"

if [ -z "$asset_url" ]; then
  echo "release asset not found for ${os}/${arch}" >&2
  exit 1
fi
if [ -z "$checksums_url" ]; then
  echo "checksums.txt not found in release assets" >&2
  exit 1
fi

asset_file="$TMP_DIR/$(basename "$asset_url")"
checksum_file="$TMP_DIR/checksums.txt"
echo "downloading $(basename "$asset_file")..."
curl -fsSL "$asset_url" -o "$asset_file"
curl -fsSL "$checksums_url" -o "$checksum_file"

expected="$(awk -v n="./$(basename "$asset_file")" '$2==n{print $1}' "$checksum_file" | head -n 1)"
if [ -z "$expected" ]; then
  expected="$(awk -v n="$(basename "$asset_file")" '$2==n{print $1}' "$checksum_file" | head -n 1)"
fi
if [ -z "$expected" ]; then
  echo "checksum entry not found for $(basename "$asset_file")" >&2
  exit 1
fi

if command -v shasum >/dev/null 2>&1; then
  actual="$(shasum -a 256 "$asset_file" | awk '{print $1}')"
elif command -v sha256sum >/dev/null 2>&1; then
  actual="$(sha256sum "$asset_file" | awk '{print $1}')"
else
  echo "missing checksum tool (shasum or sha256sum)" >&2
  exit 1
fi

if [ "$expected" != "$actual" ]; then
  echo "checksum mismatch for $(basename "$asset_file")" >&2
  echo "expected: $expected" >&2
  echo "actual:   $actual" >&2
  exit 1
fi
echo "checksum verified."

tar -xzf "$asset_file" -C "$TMP_DIR"
src_bin="$TMP_DIR/atrakta"
if [ ! -f "$src_bin" ]; then
  src_bin="$(find "$TMP_DIR" -maxdepth 2 -type f -name 'atrakta-*' | head -n 1 || true)"
fi
if [ -z "$src_bin" ] || [ ! -f "$src_bin" ]; then
  echo "extracted binary not found" >&2
  exit 1
fi

mkdir -p "$INSTALL_DIR"
target_bin="$INSTALL_DIR/$INSTALL_NAME"
if command -v install >/dev/null 2>&1; then
  install -m 0755 "$src_bin" "$target_bin"
else
  cp "$src_bin" "$target_bin"
  chmod 0755 "$target_bin"
fi

if [ "$os" = "darwin" ] && command -v xattr >/dev/null 2>&1; then
  xattr -d com.apple.quarantine "$target_bin" >/dev/null 2>&1 || true
fi

# Install `atr` as a symlink alias
alias_bin="$INSTALL_DIR/atr"
ln -sf "$target_bin" "$alias_bin"

hash -r >/dev/null 2>&1 || true

echo
echo "installed: $target_bin"
echo "alias:     $alias_bin -> $target_bin"
if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
  echo "warning: $INSTALL_DIR is not in PATH"
  echo "add this line to your shell rc and restart shell:"
  echo "  export PATH=\"$INSTALL_DIR:\$PATH\""
fi
echo
echo "next:"
echo "  atrakta --help"
echo "  atr init --interfaces cursor   # short alias"
