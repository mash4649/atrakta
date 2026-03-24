# Inspect / Preview / Simulate 契約

## 標準リゾルバ出力

すべてのリゾルバ出力に次を含める:

- input
- decision
- reason
- evidence
- next_allowed_action

## Inspect の要件

- レイヤ境界の inspect
- ガイダンス強度の inspect
- 投影適格性の inspect
- 厳格ライフサイクルの inspect
- レガシー進行の inspect
- 拡張順序の検証

## Preview の要件

- 管理スコープの preview
- 変更計画の preview
- 監査保持のドライラン preview

## Simulate の要件

- 失敗ルーティングの simulate
- ポリシー衝突の simulate

inspect 可能性は任意ではなく、デフォルトで各リゾルバ契約の一部である。

## バンドルスキーマ

- CLI 入力バンドル: `schemas/operations/bundle-input.schema.json`
- CLI 出力バンドル: `schemas/operations/bundle-output.schema.json`
- フィクスチャレポート: `schemas/operations/fixtures-report.schema.json`

## アーティファクトのエクスポート

- `--artifact-dir` は標準出力の挙動を変えずに JSON スナップショットを書き出す
- `inspect`、`preview`、`simulate` は `*.bundle.json` をエクスポートする
- `export-snapshots` はオンボーディング注入版も追加でエクスポートする:
  - `inspect.onboard.bundle.json`
  - `preview.onboard.bundle.json`
  - `simulate.onboard.bundle.json`
- `run-fixtures` は `fixtures.report.json` をエクスポートする
