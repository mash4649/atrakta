# Security Policy

## Supported Versions

| Version | Supported |
|---|---|
| 0.14.x | ✅ Active support |
| 0.13.x | ⚠️ Security fixes only |
| < 0.13 | ❌ Not supported |

## Reporting a Vulnerability

セキュリティの脆弱性を発見した場合は、**公開 Issue ではなく** 以下の方法で報告してください。

**報告先:** GitHub の [Security Advisories](https://github.com/mash4649/atrakta/security/advisories/new) を使用してください。

報告内容に含めてほしいもの:

- 脆弱性の概要
- 再現手順
- 影響範囲（どのバージョン・どの機能）
- 可能であれば修正案

## 対応プロセス

1. 報告受領後 48 時間以内に確認の返信をします
2. 修正のタイムラインをお伝えします
3. 修正リリース後、CHANGELOG に記載します（希望があればクレジットします）
4. 修正完了まで公開しないようお願いします

## Atrakta のセキュリティ設計

Atrakta はセキュリティを設計の中心に置いています:

- **セーフティコントラクト** — AI が変更できるファイルを宣言的に制限
- **追記専用イベントログ** — 操作の改ざんを防ぐハッシュチェーン
- **managed-only ガード** — 明示的な承認なしに破壊的変更を防止
- **監査証跡** — すべての AI 操作を記録・再現可能

ご協力ありがとうございます。
