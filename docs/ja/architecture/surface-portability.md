# サーフェス移植性

## 目標

Atrakta v1 は IDE 横断の同一 UX パリティを追いません。
ネイティブな UX の差は許容しつつ、意味論上の契約・安全判断・停止条件を
サーフェス間で同一に保証します。

## 非目標

- IDE 横断で同一のプロンプトやレイアウトを強制すること
- CLI と IDE で一つの対話モデルに押し込めること
- 既存のサーフェスファイルをすべて直接再生成すること

## v1 のターゲット語彙

- `agents_md`
- `ide_rules`
- `repo_docs`
- `skill_bundle`

## v1 のソース語彙

- `canonical_policy`
- `workflow_binding`
- `skill_asset`
- `repo_docs`
- `agents_md`
- `ide_rules`

`workflow_bundle` と `runtime_hook` は v1 では移植性のターゲットではありません。

## 所有権

- 既存の `AGENTS.md`、IDE ルール、リポジトリドキュメントは、まずアドバイザリ読み取りが前提
- 意味のソース・オブ・トゥルースは正規ストアのまま
- v1 は正規状態からサーフェスファイルを再生成しない
- 管理付き書き込みは `.atrakta/generated/**` に限定される

## 解決契約

`resolve_surface_portability(input) -> portability_decision`

決定には次が含まれます:

- `supported_targets[]`
- `degraded_targets[]`
- `unsupported_targets[]`
- `ingest_plan[]`
- `projection_plan[]`
- `portability_status`

## 標準的な劣化

デフォルトの劣化ポリシーは `proposal_only` です。

- 劣化または非対応の移植性を黙って成功扱いにしない
- inspect と preview は継続しうる
- apply は無効
- `run` は `next_allowed_action=propose` を伴う移植性メタデータを返す

`BLOCK` は、別途ポリシーまたは整合性違反用に予約されます。
