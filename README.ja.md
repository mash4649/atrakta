# Atrakta 正規モデル v0

バージョン: `1.0.0-alpha.2`

このリポジトリは Atrakta の新しい書き換えベースラインです。
設計は契約ファーストかつリゾルバファーストです。
機械可読な契約は `.atrakta/contract.json` に置かれます。
本ラインはリフレッシュ用トラックであり、`v0.14.1` 互換トラックではありません。

**英語版:** [README.md](README.md)  
**ドキュメント（日本語）:** [docs/ja/README.md](docs/ja/README.md)

## 設計目標

- 決定的な挙動のもとでの最大速度
- デフォルトで高い信頼性と高い安全性
- 最小の初期フットプリントと段階的な拡張
- 検出 → 提案 → 受諾 によるゼロ設定オンボーディング
- 画面間でのコンテキストのずれを許さない

## なぜ存在するか

Atrakta は、AI 支援開発を実リポジトリで使えるものにしつつ、
毎回人手で操作しないと進まない状態にしないためにあります。

開発者にとっては、次の価値を提供します。

- 書き込み前に安全な提案を出す
- セッションをまたいで run / resume を再現可能にする
- state / audit / projection の境界を明示する
- CLI / IDE / CI / 配布ツール間のつなぎ込みを減らす
- 検出・修復・適用を扱う共通契約を持つ

## 五本柱

- `schemas/`: 契約のソース・オブ・トゥルース
- `canonical/`: 正規状態の単一の書き込み元
- `resolvers/`: 決定的な判断ロジック
- `adapters/`: 外部バインドと投影サーフェス
- `tests/`: 契約およびリゾルバの検証

## 初回デリバリの範囲

1. レイヤ境界と所有権の定義
2. ガイダンスの強さと優先順位の定義
3. 一方向投影モデルの定義
4. 失敗ルーティングと厳格なライフサイクルの定義
5. 管理スコープと変更ポリシーの定義
6. レガシー統治と昇格ルールの定義
7. オペレーション能力モデルの定義
8. 拡張境界と評価順序の定義
9. 監査の整合性と保持ポリシーの定義
10. inspect / preview / simulate の出力契約の定義

## スタートアップの原則

- まず検出（Detect first）
- 安全なデフォルト
- 変更は提案ファースト
- 遅延正規化（Lazy canonicalization）
- 段階的開示（Progressive disclosure）

## 次のステップ

まず `docs/architecture/run-contract.md`（[日本語版](docs/ja/architecture/run-contract.md)）から読み、続けて
`docs/plan/v0-execution-plan.md`（[日本語版](docs/ja/plan/v0-execution-plan.md)）をイシュー順に実行します。

## クイックスタート

- `go run ./cmd/atrakta run --project-root . --json`
- `go run ./cmd/atrakta run --project-root . --non-interactive --json`
- `go run ./cmd/atrakta run --project-root . --apply --approve --json`

## インストール

- curl インストーラ（Linux / macOS）:
  - `curl -fsSL https://raw.githubusercontent.com/mash4649/atrakta/main/scripts/install.sh | bash`
- 直接ダウンロード:
  - `https://github.com/mash4649/atrakta/releases/latest`
- Docker（公開後）:
  - `docker run --rm ghcr.io/mash4649/atrakta:latest --help`

## リフレッシュの位置づけ

- 運用インターフェースはまず `run` を優先する。
- レガシーのコマンド／データのパリティは意図的にスコープ外とする。
- 評価は概念カバレッジ、決定的リプレイ、安全性不変条件に基づく。

## ドキュメントの入口

- [docs/ja/README.md](docs/ja/README.md)（日本語インデックス）
- [docs/README.md](docs/README.md)（英語インデックス）
