# 03. Parity検証

[English](../../en/04_quality/03_parity_verification.md) | [日本語](./03_Parity検証.md)

## 対象

Parity検証は interface 間の contract projection 一貫性と実行挙動整合を対象にします。

## 現行ベースライン

まずは既存検証ループを基準とします。

- `./scripts/dev/verify_loop.sh`
- `./scripts/dev/verify_perf_gate.sh`

## 追加予定

parity バックログで以下を追加予定です。

- projection drift 検知
- render hash mismatch 検知
- managed block corruption 検知
- parity 診断レポート

## 出力要件

blocking issue と warning を区別し、機械可読出力を提供すること。
