# Development Guide

Atrakta のコードベースに貢献するための開発ガイドです。

## クイックスタート

```bash
git clone https://github.com/mash4649/atrakta.git
cd Atrakta
go build ./cmd/atrakta
./atrakta --version
```

---

## アーキテクチャ概要

Atrakta は **Detect → Plan → Apply → Gate** の決定論的パイプラインで動作します。

```
ユーザー / AI ツール
        │
        ▼
  [Detect]  ← プロジェクトの現在状態を検出
        │
        ▼
  [Plan]    ← タスク DAG を生成
        │
        ▼
  [Apply]   ← トポロジー順でタスクを実行
        │
        ▼
  [Gate]    ← 安全性・品質を検証
        │
        ▼
  .atrakta/ に状態を書き込む
```

---

## internal/ パッケージ構成

| パッケージ | 役割 |
|---|---|
| `detect/` | プロジェクト状態の検出 |
| `plan/` | タスク DAG の生成 |
| `apply/` | タスクの実行（トポロジー順） |
| `gate/` | 安全性・品質ゲート |
| `contract/` | セーフティコントラクトの読み込み・検証 |
| `editsafety/` | ファイル変更の安全性チェック |
| `policy/` | 変更ポリシーの評価 |
| `events/` | イベントログの書き込み・読み込み |
| `state/` | セッション状態の管理 |
| `checkpoint/` | チェックポイントの保存・復元 |
| `progress/` | タスク進捗の追跡 |
| `taskgraph/` | タスクグラフの管理 |
| `adapter/` | インターフェースアダプター基底 |
| `ide/` | IDE 向けアダプター |
| `ifaceauto/` | インターフェース自動検出 |
| `model/` | データモデル定義 |
| `core/` | コア共通処理 |
| `context/` | 実行コンテキスト |
| `platform/` | OS 固有処理 |
| `util/` | ユーティリティ |
| `doctor/` | 状態診断 |
| `gc/` | ガベージコレクション |
| `gitauto/` | Git 自動操作 |
| `hooks/` | フック処理 |
| `migrate/` | マイグレーション |
| `bootstrap/` | 初期化処理 |
| `registry/` | コンポーネントレジストリ |
| `repomap/` | リポジトリマッピング |
| `routing/` | コマンドルーティング |
| `runtimecache/` | ランタイムキャッシュ |
| `runtimeobs/` | ランタイム可観測性 |
| `startfast/` | 高速起動最適化 |
| `subworker/` | サブワーカー処理 |
| `syncpolicy/` | 同期ポリシー |
| `wrapper/` | 外部コマンドラッパー |
| `proof/` | 証明・検証処理 |

---

## テスト

### 基本テスト

```bash
go test ./...
```

### 特定パッケージのテスト

```bash
go test ./internal/detect/...
go test ./internal/plan/...
go test ./internal/apply/...
```

### 詳細出力

```bash
go test -v ./...
```

### カバレッジ確認

```bash
go test -cover ./...
```

---

## ローカルビルド

```bash
# バイナリをビルド
go build -o atrakta ./cmd/atrakta

# 直接実行
go run ./cmd/atrakta init --interfaces cursor
go run ./cmd/atrakta doctor
```

---

## コード品質

PR を送る前に以下を実行してください:

```bash
go fmt ./...
go vet ./...
go test ./...
```

---

## 新しいコマンドを追加する手順

1. `internal/` に新しいパッケージを作成
2. `cmd/atrakta/main.go` にコマンドを登録
3. テストを `internal/<package>/<package>_test.go` に追加
4. `docs/en/02_spec/01_cli_spec.md` にコマンド仕様を追記

---

## 新しいインターフェースアダプターを追加する手順

1. `internal/adapter/` の既存アダプターを参考にする
2. `internal/ide/` に新しいアダプターを作成
3. `internal/ifaceauto/` に登録
4. `docs/en/02_spec/05_interface_matrix.md` に追記

---

## デバッグ

詳細ログを有効にする:

```bash
atrakta -v start --interfaces cursor
```

イベントログを確認:

```bash
cat .atrakta/events.jsonl
```

状態を診断:

```bash
atrakta doctor
```

---

## リリース手順

`main` ブランチへの push で自動リリースされます（`.github/workflows/release.yml`）。

手動ビルド:

```bash
./scripts/build/build_release_artifacts.sh
```

---

## 参考ドキュメント

- [CLI 仕様](docs/en/02_spec/01_cli_spec.md)
- [実行フロー](docs/en/02_spec/02_execution_flow.md)
- [データモデル](docs/en/02_spec/03_data_model.md)
- [同期ポリシー](docs/en/02_spec/04_sync_policy.md)
