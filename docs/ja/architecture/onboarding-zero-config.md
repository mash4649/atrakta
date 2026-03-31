# ゼロ設定オンボーディング

## 意図

手動設定なしで安全に始め、その後段階的に制御サーフェスを開示する。

## コアルール

- まず検出（detect first）
- 安全なデフォルト
- 変更は提案ファースト
- 遅延正規化
- 段階的開示

## モード

- `new_project`
- `brownfield_project`

モードは信頼度付きで自動推論する。
信頼度が低いときだけ確認を求める。

## 検出対象

- `agents_md` および `ide_rules`
- `workflow_binding`
- ランタイムおよびツール設定
- テストおよびスクリプトのサーフェス
- 外部送信や破壊的操作など、リスク候補

## 初期の安全デフォルト

- 読み取りのみ許可
- ローカル書き込みは提案のみ
- 破壊的操作は拒否
- 外部送信は拒否
- 未知の能力は厳格扱い
- 未マップのガイダンスはアドバイザリのみ

## オンボーディング提案バンドル

最低限のフィールド:

- `detected_assets`
- `detected_risks`
- `inferred_mode`
- `inferred_managed_scope`
- `inferred_capabilities`
- `inferred_guidance_strength`
- `inferred_default_policy`
- `inferred_failure_routing`
- `conflicts`
- `suggested_next_actions`

`inferred_failure_routing` は、オンボーディングの衝突を厳格トリガーにマッピングし、
`resolve_failure_tier` で評価して算出します。
`detected_risks` は `workflow_binding`／スクリプト／パッケージのサーフェスから、
外部送信・破壊的スクリプト・秘密露出の候補を拾います。

## 検出・推論 API

- `detect_project_root`
- `detect_mode`
- `detect_assets`
- `infer_managed_scope`
- `infer_capabilities`
- `infer_guidance_strength`
- `infer_default_policy`
- `build_onboarding_proposal`

## CLI

- `go run ./cmd/atrakta run --project-root . --json`
- `go run ./cmd/atrakta run --project-root . --non-interactive --json`
- `go run ./cmd/atrakta run --project-root . --apply --approve --json`
- `go run ./cmd/atrakta onboard --project-root . --artifact-dir .tmp/atrakta-artifacts`
- `go run ./cmd/atrakta inspect --onboard-root . --artifact-dir .tmp/atrakta-artifacts`
