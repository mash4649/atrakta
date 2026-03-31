# 管理スコープと変更ポリシー

## スコープの分類

- managed_block
- managed_include
- generated_projection
- unmanaged_user_region
- proposal_patch_only

## 変更ポリシー

- managed_block: 判断エンベロープ付きで管理付き apply を許可
- managed_include: 追記／include を優先
- generated_projection: 生成物の出力に限り置換を許可
- unmanaged_user_region: 暗黙の変更は禁止
- proposal_patch_only: 提案のみ、明示承認が必要

`repo_docs`: 追記／include を優先。
ツール設定: include／提案のみを優先。
`canonical_policy` ストア: 管理付きパス以外での置換は不許可。
既存のユーザールール: 曖昧さは提案のみにフォールバック。

## 移行の原則

移行中は二重読み取り・単一書き込みを用いる。
生成アーティファクトから正規ストアへの逆方向の自動同期は行わない。

## リゾルバ API

`check_mutation_scope(target) -> scope_decision`
