# オペレーション能力モデル

## アクションクラス

- inspect_only
- propose_only
- apply_mutation

## 正規の能力（Canonical Capabilities）

- inspect_health
- inspect_drift
- inspect_parity
- inspect_integration
- propose_repair
- apply_repair

## レガシー別名マッピング

- doctor -> inspect_health
- parity -> inspect_parity
- integration -> inspect_integration
- repair -> propose_repair

## 失敗ティアの上限

- BLOCK -> inspect_only
- PROPOSAL_ONLY -> propose_only
- DEGRADE_TO_STRICT -> propose_only
- WARN_ONLY -> apply_mutation

実効アクションクラスが `apply_mutation` でない限り、暗黙の変更は禁止。
