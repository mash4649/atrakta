# レイヤ境界契約

## 目標

Core、Canonical、Extension の各レイヤ間の所有権を固定する。

## レイヤごとの許可と禁止

### Core 契約

許可:

- Request
- Decision
- Result
- Error

禁止:

- ランタイムプロファイル
- アセット参照
- 派生投影本文
- 投影ソースとしてのタスク状態
- 判断ソースとしての監査イベント

### Canonical ストア

許可:

- Capability
- `canonical_policy`
- Task State
- Audit Event

禁止:

- 派生投影本文
- ランタイム専用の一時状態
- 拡張の実行可能アセットへの直接アクセス

### Extension アセット

許可:

- ランタイムプロファイル
- `repo_docs`
- `skill_asset`
- `workflow_binding`
- Provenance

禁止:

- `canonical_policy` の直接上書き
- コア契約への直接変更

## 所有者マッピング

- Request -> core
- Decision -> core
- Result -> core
- Error -> core
- Capability -> canonical
- `canonical_policy` -> canonical
- Task State -> canonical
- Audit Event -> canonical
- Runtime Profile -> extension
- `repo_docs` -> extension
- `skill_asset` -> extension
- `workflow_binding` -> extension
- Provenance -> extension

## レイヤ API

`classify_layer(item) -> core | canonical | extension`

入力要件:

- `item.kind`
- `item.schema_id`

出力要件:

- `decision` は `core`、`canonical`、`extension` のいずれか
- `reason`
- `evidence`

## 禁止ルール

レイヤの所有権をまたぐアイテムは、変更時に拒否し、inspect 出力で報告しなければならない。
