# 04. Extension検証

[English](../../en/04_quality/04_extension_verification.md) | [日本語](./04_Extension検証.md)

## 対象

MCP / plugin / skill / workflow / hook の projection 整合性と互換性検証を対象にします。

## 現行ベースライン

専用 verify スクリプト導入前は、既存 CI 検証を回帰基準として使用します。

## 追加予定

- extension projection 完全性チェック
- unsupported warning 検証（silent ignore しない）
- brownfield append/include の idempotency 検証
- projection repair の managed-only 範囲検証

## 期待特性

- canonical contract から deterministic に出力される
- unsupported projection は warning で必ず可視化される
- user-managed 領域を破壊的に上書きしない
