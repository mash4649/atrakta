#!/usr/bin/env bash
set -euo pipefail

if ! command -v brew >/dev/null 2>&1; then
  printf >&2 'brew is required for this installer\n'
  exit 1
fi

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
export ATRAKTA_INSTALL_DIR="$(brew --prefix)/bin"
export ATRAKTA_OS="darwin"
exec "$script_dir/install.sh" "$@"
