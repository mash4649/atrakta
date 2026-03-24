# ガイダンスの強さと優先順位

## 強さのクラス

- `authoritative_constraint`
- `orchestration_constraint`
- `executable_guidance`
- `advisory_map`
- `tool_hint`

## サーフェスのクラス

- `decision`
- `orchestration`
- `mutation`
- `projection`
- `diagnostics`

## マッピング

- `canonical_policy` -> authoritative_constraint; decision, mutation
- `workflow_binding` -> orchestration_constraint; orchestration, mutation
- `skill_asset` -> executable_guidance; orchestration, mutation
- `repo_docs` -> advisory_map; projection, diagnostics
- `ide_rules` -> tool_hint; diagnostics

## 優先ルール

1. `canonical_policy`
2. `workflow_binding`
3. `skill_asset`
4. `repo_docs`
5. `ide_rules`

`canonical_policy` は常に勝つ。
参照アセットは、`canonical_policy` にマップされない限りアドバイザリである。
`repo_docs` と `ide_rules` は判断や承認を上書きできない。
未マップのレガシーガイダンスはアドバイザリのみ。

## リゾルバ API

`resolve_guidance_precedence(set) -> ordered_list`

出力には順位付け、衝突理由、次に許可されるアクションが含まれる。

`agents_md` や `ide_rules` などガイダンス担体の移植性については
[サーフェス移植性](surface-portability.md) を参照。
