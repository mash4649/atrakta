# Plugin SDK インターフェース定義

Plugin SDK は拡張開発者向けの宣言的な契約です。
ランタイムローダや実行プロトコルは定義しません。
Atrakta が検査・検証・順序付けできる plugin マニフェストの形を定義します。

## バージョン

- `schema_version`: `plugin-sdk.v1`

## 必須フィールド

- `id`: 安定した plugin 識別子
- `name`: 人間向けの名前
- `kind`: `plugin` 固定
- `targets`: 以下の 1 つ以上
  - `adapter`
  - `projection`
  - `operations`
- `entrypoint`: host が plugin を呼び出す方法
- `capabilities`: 宣言された plugin の能力
- `permissions`: 安全制約

## 安全制約

- plugin は core contract を直接変更してはいけない。
- plugin は canonical state を直接書き込んではいけない。
- plugin は host が明示的に許可した制御アクション以外では read-only / advisory として扱う。
- `can_mutate_core_contract` と `can_write_canonical` は常に `false`。

## エントリポイント

- `binary`: 実行可能ファイルのパスまたはコマンド
- `go-package`: host が別レイヤー統合で読み込む Go パッケージ参照

## 例

```json
{
  "schema_version": "plugin-sdk.v1",
  "id": "projection-html",
  "name": "HTML Projection Plugin",
  "kind": "plugin",
  "targets": ["projection"],
  "entrypoint": {
    "type": "binary",
    "value": "./bin/projection-html"
  },
  "capabilities": ["render-html", "preview-static"],
  "permissions": {
    "can_mutate_core_contract": false,
    "can_write_canonical": false,
    "can_block_execution": false
  },
  "metadata": {
    "description": "Canonical projection output を HTML として描画する。"
  }
}
```

## 拡張境界との関係

- 拡張境界は plugin が参加できる位置を決める。
- Plugin SDK はその参加対象を宣言する契約である。
- [拡張境界と評価順序](extension-boundary.md) を参照。
