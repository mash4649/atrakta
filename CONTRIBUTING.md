# Contributing to Atrakta

コントリビューションを歓迎します。バグ修正・機能追加・ドキュメント改善、どんな形でも助かります。

## はじめる前に

### 環境要件

- Go 1.26 以上
- Git

### リポジトリのクローン

```bash
git clone https://github.com/afwm/Atrakta.git
cd Atrakta
```

### ビルドと確認

```bash
go build ./cmd/atrakta
./atrakta --version
```

### テスト実行

```bash
go test ./...
```

---

## コントリビューションの種類

### バグ報告

1. [Issues](https://github.com/afwm/Atrakta/issues) を検索して重複がないか確認
2. 「Bug Report」テンプレートで Issue を作成
3. 再現手順・環境・期待する動作を明記

### 機能提案

1. [Discussions](https://github.com/afwm/Atrakta/discussions) で先に議論
2. 合意が取れたら「Feature Request」テンプレートで Issue を作成

### ドキュメント改善

- `docs/en/` または `docs/ja/` 以下のファイルを直接編集して PR を送ってください
- 誤字・リンク切れ・説明の追加など、小さな修正でも歓迎です

### コード変更

1. 対応する Issue を確認（なければ作成）
2. `good-first-issue` ラベルの Issue が入門に最適です
3. 以下の手順で PR を送ってください

---

## PR を送る手順

```bash
# 1. フォーク後にクローン
git clone https://github.com/YOUR_USERNAME/Atrakta.git
cd Atrakta

# 2. ブランチを作成
git checkout -b fix/issue-123

# 3. 変更を加える

# 4. フォーマット・テスト
go fmt ./...
go vet ./...
go test ./...

# 5. コミット
git commit -m "fix: describe what you fixed (#123)"

# 6. プッシュ
git push origin fix/issue-123

# 7. GitHub で PR を作成
```

---

## コードスタイル

- `gofmt` でフォーマットする（`go fmt ./...`）
- エクスポートされる関数にはコメントを書く
- 関数は小さく保つ
- テストは変更に合わせて追加・更新する

---

## マージの基準

PR がマージされるには以下が必要です:

- `go test ./...` が通ること
- `go vet ./...` でエラーがないこと
- 変更内容の説明が PR に書かれていること
- ユーザー向けの変更の場合、ドキュメントが更新されていること

---

## 開発の詳細

アーキテクチャ・内部構造・テスト方法の詳細は [DEVELOPMENT.md](./DEVELOPMENT.md) を参照してください。

---

## ライセンス

コントリビューションは Apache License 2.0 のもとで提供されます。
