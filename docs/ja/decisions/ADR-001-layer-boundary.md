# ADR-001 レイヤ境界

状態: 採用

Core には request、decision、result、error のみを含める。
Canonical には `canonical_policy`、capability、タスク状態、監査イベントを含める。
Extension にはランタイムプロファイル、`repo_docs`、`skill_asset`、`workflow_binding`、provenance を含める。
レイヤをまたぐ所有権の変更は禁止する。
