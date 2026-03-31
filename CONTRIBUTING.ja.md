# Atrakta へのコントリビュート

コントリビュートありがとうございます。このリポジトリは **契約ファースト** / **リゾルバファースト** です。
機械可読な契約は `.atrakta/contract.json` にあり、スキーマのソース・オブ・トゥルースは `schemas/` にあります。

まずは設計の前提を揃えるために、`docs/architecture/run-contract.md` と
`docs/architecture/adapter-invocation.md` を参照してください。

## 前提条件

- Go **1.26**（CI は Go 1.26 を使用）
- macOS / Linux / Windows（CI は Linux。変更はポータブルに）

新しいツール依存は追加しないでください（導入が必要な場合は別途合意の上で）。

## リポジトリ構成（最短の把握）

- `schemas/`: 契約スキーマ（ソース・オブ・トゥルース）
- `resolvers/`: 決定的な判断ロジック
- `adapters/`: 薄い invoker（`atrakta run` を呼ぶ。内部コマンド構成に依存しない）
- `canonical/`: 正規状態（単一の書き込み元）
- `fixtures/`: 決定的なフィクスチャ入力とコミット済みスナップショット
- `tests/`: 契約およびリゾルバの検証

## 開発セットアップ

リポジトリをクローンしたら、`atrakta/` ディレクトリ配下を作業ディレクトリとして実行します。

推奨するローカルセットアップ:

1. Go 1.26 をインストールする
2. リポジトリをクローンする
3. `cd atrakta`
4. ツールチェーンを確認する:

```bash
go build ./...
go test ./...
go run ./cmd/atrakta run-fixtures
go run ./cmd/atrakta verify-coverage
```

Go のビルドキャッシュを使いたい場合は `.tmp/go-build` を作って再利用してください:

```bash
mkdir -p .tmp/go-build
GOCACHE="$(pwd)/.tmp/go-build" go test ./...
```

主な入口（ローカル開発）:

- `go run ./cmd/atrakta run --project-root . --json`
- `go run ./cmd/atrakta run --project-root . --non-interactive --json`
- `go run ./cmd/atrakta run --project-root . --apply --approve --json`

注: `atrakta run` は **単一の実行プリミティブ** です。他のコマンドは実装部品／デバッグ・サーフェスです。

## テスト実行（ローカル）

全テスト:

```bash
go test ./...
```

CI のキャッシュ挙動に寄せたい場合:

```bash
mkdir -p .tmp/go-build && GOCACHE="$(pwd)/.tmp/go-build" go test ./...
```

CI と同じ順序でスナップショット／カバレッジ・ゲートを実行する場合:

```bash
go run ./cmd/atrakta run-fixtures
go run ./cmd/atrakta verify-coverage
```

スキーマ／リゾルバ／フィクスチャを触る場合は、CI ゲートもローカルで実行してください:

```bash
mkdir -p .tmp/go-build && GOCACHE="$(pwd)/.tmp/go-build" go run ./cmd/atrakta verify-coverage
```

## スナップショット・ゲート（fixtures/snapshots）

CI は固定セットの JSON アーティファクトを再生成し、`fixtures/snapshots/` にコミットされている内容と
一致しない場合に失敗します。

全スナップショット再生成（CI と同等）:

```bash
mkdir -p .tmp/atrakta-artifacts
go run ./cmd/atrakta export-snapshots --dir .tmp/atrakta-artifacts
```

コミット済みスナップショットを更新:

```bash
cp -f .tmp/atrakta-artifacts/*.json fixtures/snapshots/
```

最後に:

```bash
go test ./...
go run ./cmd/atrakta verify-coverage
```

### スナップショットが変わるべきタイミング

スナップショット変更は、基本的に以下のいずれかに限ります:

- リゾルバや契約の挙動を **意図的に** 変更した
- スキーマ変更によりエンベロープ／出力形状が変わった
- フィクスチャを意図的に追加／更新した

意図せず差分が出た場合は、まず決定性と契約整合を疑って原因を切り分けてください。

## フィクスチャの追加

フィクスチャは以下を満たすことを意図しています:

- **決定的**
- **最小**（狙いを絞った小さなケース）
- **契約の代表性**（関連するスキーマ／リゾルバ経路を確実に通す）

一般的な流れ:

1. 既存の命名規則に合わせて `fixtures/` 配下に入力を追加／更新
2. `export-snapshots` でスナップショット再生成
3. `fixtures/snapshots/` の更新をコミット
4. `go run ./cmd/atrakta verify-coverage` で明示的カバレッジ・マッピングを確認

新しいカテゴリ／出力を増やす場合は、エクスポートと検証がその対象を含むようにし、CI ゲートが決定的であることを維持してください。

## スキーマ変更ポリシー

Atrakta は契約駆動です。スキーマ変更は意図的に行い、必要に応じて以下を同時に更新します:

- フィクスチャ／スナップショット（出力形状が変わる場合）
- カバレッジ・マッピング（`verify-coverage` が通ること）
- 契約が記述されているドキュメント（必要に応じて）

指針:

- 破壊的変更よりも追加的変更（フィールド追加）を優先
- 破壊的変更の場合は理由と移行方針を明確化
- `.atrakta/contract.json` と `schemas/` の期待を整合させる

`schemas/operations/*.schema.json` やリゾルバが追加されたのに明示的カバレッジ・マッピングが無い場合、CI は失敗します。

## リゾルバ契約の期待

リゾルバは決定的なコアです。以下を崩さないことが必須です:

- **決定性**: 時刻、乱数、map の反復順、環境差分への暗黙依存は禁止（必要なら入力として明示化）
- **安定した順序**: 出力順序は意図的かつ再現可能であること
- **ポリシー境界**: managed-scope と approval 境界をバイパスしない
- **managed scope ルール**: `unmanaged_user_region` は変更しない。曖昧さがあれば proposal-only にフォールバック（`docs/architecture/managed-scope.md`）
- **承認ゲート**: 書き込みは明示的承認（`--approve` または対話承認）が必要。必要時は `NEEDS_APPROVAL` を返す
- **アダプタ契約**: アダプタは `atrakta run` を呼び、終了コード `0/1/2/3` を `docs/architecture/adapter-invocation.md` に従って扱う

リゾルバの挙動を変更した場合は、フィクスチャ／スナップショット更新と、run 契約に沿った portability メタデータ／終了コードの整合を確認してください。

## PR チェックリスト

- [ ] `go build ./...` が通る
- [ ] `go test ./...` が通る
- [ ] `go run ./cmd/atrakta run-fixtures` が通る
- [ ] `go run ./cmd/atrakta verify-coverage` が通る（schemas/resolvers/fixtures を触る場合）
- [ ] 意図的な変更に応じてスナップショットを更新しコミットした
- [ ] 契約変更があればドキュメントも更新した
