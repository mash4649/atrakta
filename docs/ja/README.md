# Atrakta ドキュメント（日本語）

利用方法と仕様ドキュメントの入口です。

**英語版:** [../README.md](../README.md)

## まず読む

- [実行契約（run）](architecture/run-contract.md)
- [アダプタ呼び出し契約](architecture/adapter-invocation.md)
- [サーフェス移植性](architecture/surface-portability.md)
- [概念カバレッジマトリクス](plan/concept-coverage-matrix.md)
- [ゼロ設定オンボーディング](architecture/onboarding-zero-config.md)
- [実行計画 v0](plan/v0-execution-plan.md)
- [実装状況](plan/implementation-status.md)
- [Inspect / Preview / Simulate 契約](architecture/inspectability-contract.md)

## 使い方

主な CLI:

- `go run ./cmd/atrakta run --project-root . --json`
- `go run ./cmd/atrakta run --project-root . --non-interactive --json`
- `go run ./cmd/atrakta run --project-root . --apply --approve --json`

レガシー／デバッグ用の入口:

- `go run ./cmd/atrakta onboard --project-root . --artifact-dir .tmp/atrakta-artifacts`
- `go run ./cmd/atrakta inspect --onboard-root . --artifact-dir .tmp/atrakta-artifacts`
- `go run ./cmd/atrakta preview --onboard-root . --artifact-dir .tmp/atrakta-artifacts`
- `go run ./cmd/atrakta simulate --onboard-root . --artifact-dir .tmp/atrakta-artifacts`
- `go run ./cmd/atrakta accept --project-root . --artifact-dir .tmp/atrakta-artifacts`
- `go run ./cmd/atrakta mutate inspect --target .atrakta/generated/repo-map.generated.json`
- `go run ./cmd/atrakta mutate propose --target .atrakta/generated/repo-map.generated.json --content '{"k":"v"}'`
- `go run ./cmd/atrakta mutate apply --project-root . --target .atrakta/generated/repo-map.generated.json --content-file patch.json --allow`
- `go run ./cmd/atrakta audit append --action manual_check --level A2`
- `go run ./cmd/atrakta audit verify --level A2`
- `go run ./cmd/atrakta doctor --execute`
- `go run ./cmd/atrakta parity --execute`
- `go run ./cmd/atrakta integration --execute`
- `go run ./cmd/atrakta extensions --project-root .`
- `go run ./cmd/atrakta run-fixtures --artifact-dir .tmp/atrakta-artifacts`
- `go run ./cmd/atrakta export-snapshots --dir fixtures/snapshots`
- `go run ./cmd/atrakta verify-coverage`

## 仕様マップ

- [レイヤ境界契約](architecture/layer-boundary.md)
- [ガイダンスの強さと優先順位](architecture/guidance-precedence.md)
- [投影モデルと一方向ルール](architecture/projection-model.md)
- [失敗ルーティングと厳格ライフサイクル](architecture/failure-routing.md)
- [管理スコープと変更ポリシー](architecture/managed-scope.md)
- [レガシー統治と昇格ルール](architecture/legacy-governance.md)
- [オペレーション能力モデル](architecture/operations-capability.md)
- [拡張境界と評価順序](architecture/extension-boundary.md)
- [監査の整合性と保持ポリシー](architecture/audit-integrity.md)
- [サーフェス移植性](architecture/surface-portability.md)

## 計画・補足

- [用語整合のフォローアップ](plan/follow-up-vocabulary-alignment.md)
- [v0 をハーネス後継とする際のギャップ分析](plan/harness-successor-gaps.md)

## アーキテクチャ決定記録（ADR）

- [ADR-001 レイヤ境界](decisions/ADR-001-layer-boundary.md)
- [ADR-002 ガイダンスの強さ](decisions/ADR-002-guidance-strength.md)
- [ADR-003 一方向投影](decisions/ADR-003-one-way-projection.md)

## アーティファクト

- `schemas/` に契約定義がある。
- `fixtures/` に決定的なフィクスチャ入力とスナップショットがある。
- `operations/README.md` にランタイム向けオペレーションとスナップショット方針が書かれている（英語）。
- `tests/` に契約およびリゾルバの検証がある。

## サンプル JSON（英語コメント・キーは原文）

例のペイロードはスキーマ整合のため [../examples/](../examples/) を参照（ファイル内容は英語キーのまま）。
