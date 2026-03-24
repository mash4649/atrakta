# 投影モデルと一方向ルール

## 投影の適格性

- 許可: `canonical_policy`、`repo_docs`、`skill_asset`、`workflow_binding`
- 条件付き: Decision、Result
- 禁止: Task State、Audit Event

条件付きソース単体では判断の根にはできない。
投影はランタイムの一時状態に依存してはならない。

## 投影の種類

- `durable`
- `ephemeral`
- `diagnostics`

## 評価順序

1. 正規（canonical）を最初に
2. 次にオーバーレイ
3. 次に include
4. 最後に投影レンダリング

## 一方向ルール

オーバーレイおよび投影の出力は、正規ストアへ自動書き戻しできない。
逆方向の同期は提案のみ。

## リゾルバ API

`check_projection_eligibility(source) -> allowed | conditional | forbidden`
