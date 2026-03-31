# 失敗ルーティング

## 失敗クラス

- policy_failure
- approval_failure
- capability_resolution_failure
- projection_failure
- adapter_execution_failure
- provenance_failure
- audit_integrity_failure
- legacy_conflict_failure
- surface_portability_failure

## ティアの種類

- BLOCK
- DEGRADE_TO_STRICT
- PROPOSAL_ONLY
- WARN_ONLY

## 必須マッピング

各失敗クラスは次を定義する:

- default_tier
- can_override
- requires_human_review

フェイルクローズは常にブロックと同義ではない。
実行停止と投影停止は別制御である。
診断用投影の失敗と実行の失敗は別ルートで扱う。

## リゾルバ API

`resolve_failure_tier(failure_class, context) -> tier_decision`

## オンボーディングとの接続

ゼロ設定オンボーディングは、検出された衝突を厳格トリガーにマッピングし、
`resolve_failure_tier` を通じて `inferred_failure_routing` として扱う。
