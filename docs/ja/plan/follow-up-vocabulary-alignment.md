# フォローアップメモ: 用語の整合

## 背景

アーキテクチャドキュメントは `docs/architecture/surface-portability.md` で定義された
語彙に揃えられた。本メモは、アーキテクチャ以外のドキュメントに残る
フォローアップ作業を追跡する。

## フォローアップタスク

1. 例ペイロードの語彙を揃える
   - 対象: `docs/examples/sample-onboarding-proposal.json`
   - レガシーなサーフェス名（例: `AGENTS.md`）のキー／値を、
     正規の移植性語彙（`agents_md`、`ide_rules`、`repo_docs`、`skill_bundle`）で
     表すべきか確認する。
   - 例のフィールドを変える前にスキーマ互換を確認する。

2. ドキュメント横断の用語スイープ
   - `docs/README.md`、`docs/plan/*.md`、`docs/decisions/*.md` を再スキャンし、
     `Policy`、`Workflow`、`Skill`、`Repo Map`、`IDE rules` などのレガシーラベルを洗い出す。
   - 必要なら人が読むファイル名参照は残しつつ、概念名は正規の語彙ラベルを優先する。

3. コントリビュータ向けの語彙注記を追加する
   - 将来の表記ゆれを減らすため、ドキュメント入口（例: `docs/README.md`）に
     短い「推奨用語」セクションを追加する。

## 完了条件

- アーキテクチャ以外のドキュメントも `surface-portability.md` と同じ概念ラベルを使う。
- 例の JSON は現行スキーマとテストに対して有効なままである。
- 少なくとも一つのコントリビュータ向けドキュメントに用語ガイドが書かれている。
