#!/usr/bin/env bash
set -euo pipefail

repo="${ATRAKTA_REPO:-mash4649/atrakta}"
version="${ATRAKTA_VERSION:-latest}"
install_dir="${ATRAKTA_INSTALL_DIR:-$HOME/.local/bin}"
bin_name="${ATRAKTA_BIN_NAME:-atrakta}"
os="${ATRAKTA_OS:-$(uname -s | tr '[:upper:]' '[:lower:]')}"
arch="${ATRAKTA_ARCH:-$(uname -m)}"

normalize_arch() {
  case "$1" in
    x86_64|amd64) printf 'amd64' ;;
    arm64|aarch64) printf 'arm64' ;;
    *)
      printf >&2 'unsupported architecture: %s\n' "$1"
      exit 1
      ;;
  esac
}

normalize_os() {
  case "$1" in
    linux|darwin) printf '%s' "$1" ;;
    *)
      printf >&2 'unsupported operating system: %s\n' "$1"
      exit 1
      ;;
  esac
}

version_tag() {
  case "$version" in
    latest) printf 'latest' ;;
    v*) printf '%s' "$version" ;;
    *) printf 'v%s' "$version" ;;
  esac
}

release_url() {
  if [[ "$version" == "latest" ]]; then
    printf 'https://github.com/%s/releases/latest/download/%s' "$repo" "$1"
    return
  fi
  printf 'https://github.com/%s/releases/download/%s/%s' "$repo" "$(version_tag)" "$1"
}

os="$(normalize_os "$os")"
arch="$(normalize_arch "$arch")"
archive_name="${bin_name}_${os}_${arch}.tar.gz"
download_url="$(release_url "$archive_name")"
tmp_dir="$(mktemp -d)"
cleanup() {
  rm -rf "$tmp_dir"
}
trap cleanup EXIT

if ! command -v curl >/dev/null 2>&1; then
  printf >&2 'curl is required to install atrakta\n'
  exit 1
fi

if ! command -v tar >/dev/null 2>&1; then
  printf >&2 'tar is required to install atrakta\n'
  exit 1
fi

mkdir -p "$tmp_dir"
archive_path="$tmp_dir/$archive_name"
printf 'downloading %s\n' "$download_url"
curl -fsSL "$download_url" -o "$archive_path"
tar -xzf "$archive_path" -C "$tmp_dir"

binary_path="$tmp_dir/$bin_name"
if [[ ! -f "$binary_path" ]]; then
  binary_path="$(find "$tmp_dir" -type f \( -name "$bin_name" -o -name "$bin_name.exe" \) | head -n 1)"
fi
if [[ -z "${binary_path:-}" || ! -f "$binary_path" ]]; then
  printf >&2 'installed archive does not contain %s\n' "$bin_name"
  exit 1
fi

mkdir -p "$install_dir"
install_path="$install_dir/$bin_name"
install -m 0755 "$binary_path" "$install_path"
printf 'installed %s to %s\n' "$bin_name" "$install_path"
