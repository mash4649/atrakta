# 実装状況

## 完了

- イシュー 1 ベースライン
  - レイヤ所有権の契約を文書化
  - `classify_layer` リゾルバをテスト付きで実装
- イシュー 2 ベースライン
  - ガイダンスの強さと優先順位の契約を文書化
  - `resolve_guidance_precedence` リゾルバをテスト付きで実装
- イシュー 3 ベースライン
  - 投影モデルを文書化
  - `check_projection_eligibility` リゾルバをテスト付きで実装
- イシュー 4 ベースライン
  - 失敗ルーティングと厳格ライフサイクルを文書化
  - `resolve_failure_tier` リゾルバをテスト付きで実装
  - `strict_state_machine` リゾルバをテスト付きで実装
- イシュー 5 ベースライン
  - 管理スコープと変更ポリシーを文書化
  - `check_mutation_scope` リゾルバをテスト付きで実装
- イシュー 6 ベースライン
  - レガシー統治と昇格ルールを文書化
  - `resolve_legacy_status` リゾルバをテスト付きで実装
  - `detect_legacy_drift` リゾルバをテスト付きで実装
- イシュー 7 ベースライン
  - オペレーション能力モデルを文書化
  - `resolve_operation_capability` リゾルバをテスト付きで実装
- イシュー 8 ベースライン
  - 拡張境界と評価順序を文書化
  - `resolve_extension_order` リゾルバをテスト付きで実装
- イシュー 9 ベースライン
  - 監査の整合性と保持ポリシーを文書化
  - `resolve_audit_requirements` リゾルバをテスト付きで実装
- イシュー 10 ベースライン
  - inspect／preview／simulate の出力契約を文書化
  - クロスリゾルバの出力契約テストを実装

## 進行中

- インテグレーションベースライン完了（スナップショットゲート有効）
- `atrakta run` の契約とアダプタ呼び出しドキュメントを追加
- `agents_md`、`ide_rules`、`repo_docs`、`skill_bundle` 向けの意味論移植性 v1 を追加

## 次

現在の v0 契約スコープについて、ブロックとなるベースライン作業は残っていない。

## インテグレーションの進捗

- 主要な実行プリミティブとして `atrakta run` を追加:
  - 初回受諾向けのオンボーディング経路
  - 正規あり向けの detect／plan／apply 経路
- `.atrakta/contract.json` の機械契約ドキュメントを追加
- 意味論移植性リゾルバと run ゲートを追加:
  - `adapters/bindings/*/binding.json` から能力宣言を読み込む
  - `resolve_surface_portability`
  - 劣化または非対応サーフェスでは提案のみにフォールバック
- `cmd/atrakta` 配下に CLI 入口を追加:
  - `inspect`
  - `preview`
  - `simulate`
  - `onboard`
  - `run-fixtures`
- 入力・出力バンドル向けの CLI スキーマ検証フックを追加
- `internal/onboarding` 配下にゼロ設定オンボーディング提案ビルダーを追加:
  - `detect_project_root`
  - `detect_mode`
  - `detect_assets`
  - `infer_managed_scope`
  - `infer_capabilities`
  - `infer_guidance_strength`
  - `infer_default_policy`
  - `build_onboarding_proposal`
- オンボーディング提案スキーマ検証フックを追加:
  - `schemas/operations/onboarding-proposal-bundle.schema.json`
- オンボーディングの衝突から失敗ルーティングへの接続を追加:
  - オンボーディングが `inferred_failure_routing` を出力
  - 厳格トリガ経由で `resolve_failure_tier` により導出
- オンボーディングからパイプラインへのインテグレーション経路を追加:
  - `inspect`／`preview`／`simulate --onboard-root` がオンボーディング由来の失敗コンテキストをバンドル実行に注入
- オンボーディングのリスク検出を追加:
  - パッケージ／ワークフロー／スクリプト内容スキャンから `detected_risks`
- 受諾／永続化フローを追加:
  - `accept` が `.atrakta/canonical`、`.atrakta/generated`、`.atrakta/state`、`.atrakta/audit` に書き込む
- 変更 3 フェーズのランタイムコマンド面を追加:
  - `mutate inspect|propose|apply`
- 監査整合性のランタイムコマンドを追加:
  - `audit append`
  - `audit verify`
- オペレーション別名コマンド面を追加:
  - `doctor`
  - `parity`
  - `integration`
- 拡張マニフェスト解決コマンドを追加:
  - `extensions`
- オンボーディング注入パイプラインのスナップショットを追加:
  - `inspect.onboard.bundle.json`
  - `preview.onboard.bundle.json`
  - `simulate.onboard.bundle.json`
- `--artifact-dir` による JSON アーティファクトのエクスポートモードを追加
- `internal/pipeline` 配下に順序付きリゾルバパイプラインランナーを追加
- `internal/fixtures` 配下にフィクスチャランナーを追加
- 順序付きパイプライン出力の決定的リプレイテストを追加
- フィクスチャコーパスを通過させるフィクスチャランナーテストを追加
- `go test` とスナップショットエクスポート用の GitHub Actions CI ワークフローを追加
- 必須スナップショットゲートを追加: CI が生成物を `fixtures/snapshots/*.json` と比較
- 決定的なゼロ設定推論出力のため、オンボーディング提案スナップショットを同じゲートに追加
- 次を読み込み強制するスキーマ駆動の検証フックを追加:
  - `schemas/operations/bundle-input.schema.json`
  - `schemas/operations/bundle-output.schema.json`
  - `schemas/operations/fixtures-report.schema.json`
- 次のためのカバレッジゲートコマンド `verify-coverage` を追加:
  - オペレーションスキーマのカバレッジポリシー（`schemas/operations/coverage-policy.json`）
  - リゾルバとフィクスチャの対応（`fixtures/resolver-fixture-map.json`）
- `go run ./cmd/atrakta verify-coverage` を走らせる CI ステップを追加
- 次をカバーするフィクスチャファミリを拡張:
  - `strict_state_machine`
  - `detect_legacy_drift`
- オンボーディング推論のフィクスチャカバレッジを追加:
  - `fixtures/onboarding/onboarding-proposal.fixture.json`
  - `run-fixtures` で検証し、`fixtures.report.json` スナップショットに記録
