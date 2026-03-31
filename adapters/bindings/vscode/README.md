# VS Code Binding

VS Code 向けの `wrap` / `ide-autostart` 統合の定義です。

## Manual Verification

1. `go run ./cmd/atrakta wrap install --tool vscode --dry-run --json`
2. `go run ./cmd/atrakta ide-autostart --dry-run --json`

期待する確認ポイント:
- `binding.json` が `install_path`, `script_template`, `capabilities`, `autostart_config` を持つ
- `wrap install` の `--dry-run` 出力が deterministic である
- `ide-autostart` の生成物が VS Code の `.vscode/tasks.json` と整合する
