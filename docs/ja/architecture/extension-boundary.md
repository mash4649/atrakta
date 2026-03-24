# 拡張境界と評価順序

## 第一パスの分類

- capability_adapters
- orchestration_assets
- runtime_hook

## 第二パスのマッピング

- MCP -> アダプタ、能力ソースのみ
- plugin -> アダプタ、投影、オペレーションのみ
- `skill_asset` -> アセットレイヤのみ
- `workflow_binding` -> アセットに加えオーケストレーションのみ
- `runtime_hook` -> ランタイムおよびオペレーションのライフサイクルのみ

いずれもコア契約を直接変更できない。
診断用アセットは実行ポリシーを拘束できない。
フックは正規状態を直接変えられない。

## 評価順序

1. `canonical_policy`
2. `workflow_binding`
3. `skill_asset`
4. `ide_rules`
5. `runtime_hook`
6. projection plugin

## リゾルバ API

`resolve_extension_order(set) -> ordered_extensions`
