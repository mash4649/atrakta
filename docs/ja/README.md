# Atrakta ドキュメント

[English](../en/README.md) | [日本語](./README.md)

本ディレクトリは、現行実装（Go版）の仕様・運用・品質情報を日本語で整理した正本です。

- 対象バージョン: `v0.14.1`（[VERSION](../../VERSION)）
- 最終更新日: `2026-03-11`

## 構成

- `01_全体`: 目的、設計原則、現在の提供範囲
- `02_仕様`: CLI仕様、実行フロー、データモデル、同期ポリシー、Parity Contract、Extension Surface
- `spec`: import mapping/pipeline/registry/isolation/retrieval の補助仕様
- `03_運用`: 導入手順、日常運用、トラブル対応
  - Brownfield導入
  - リリースチェックリスト（完了条件ゲート）
  - 配布手順（バイナリ配布前提）
  - GitHub Issue登録（実装バックログ一括投入）
- `04_品質`: テスト、ベンチマーク、Parity検証、Extension検証

## 先に読む順番

1. `01_全体/01_概要.md`
2. `02_仕様/01_CLI仕様.md`
3. `02_仕様/06_Parity Contract.md`
4. `02_仕様/07_Extension Surface.md`
5. `03_運用/01_導入手順.md`
6. `03_運用/07_Brownfield導入.md`
7. `03_運用/09_リリースチェックリスト.md`
8. `03_運用/10_ImportReviewRunbook.md`
9. `03_運用/08_GitHub_Issue登録.md`
10. `04_品質/03_Parity検証.md`
11. `04_品質/04_Extension検証.md`
12. `04_品質/01_検証コマンド.md`
13. `../spec/EXTERNAL_REPOSITORY_IMPORT_MAPPING_SPEC.md`
14. `../../CHANGELOG.md`

## 整理ポリシー

- 重複説明は持たず、機能ごとに1つの正本へ統合する。
- 旧文書は保持せず、必要時は Git 履歴と `../../CHANGELOG.md` を参照する。
- 実装と乖離した記述は更新時に削除し、推測記述を残さない。
