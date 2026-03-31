# Cursor Binding

Cursor 向けの `wrap` / `hook` / `ide-autostart` 統合の基点です。

## Manual Verification

1. `go run ./cmd/atrakta wrap install --tool cursor --dry-run --json`
2. `go run ./cmd/atrakta hook install --hook-type pre-commit --dry-run --json`
3. `go run ./cmd/atrakta ide-autostart --dry-run --json`

期待する確認ポイント:
- `binding.json` が `install_path`, `script_template`, `capabilities`, `autostart_config` を持つ
- `wrap install` の `--dry-run` 出力が deterministic である
- `hook` と `ide-autostart` の生成物が Cursor 向けパスに整合する
