# Extension Surface

[English](../../en/02_spec/07_extension_surface.md) | [日本語](./07_Extension Surface.md)

## Purpose

拡張要素を contract 上の一級概念として定義します。

## Extension entities

- MCP: 外部能力を明示設定で接続する server 連携。
- plugin: interface-native な拡張モジュール。
- skill: 再利用可能な作業指示・運用レシピ。
- workflow: 手順列と実行フックの定義。
- hook: shell/git/IDE/runtime ライフサイクルのイベント連携点。

## Merge strategy

- 既定: append-first
- 明示モード: `append`, `include`, `replace`

## Brownfield modes

- append mode: 既存ファイルへ managed block を追記
- include mode: include 用ファイルを生成して参照させる
- replace mode: 明示同意がある managed target にのみ許可

## Unsupported policy

- silent ignore しない
- field path と理由を warning で表示する
- native projection 非対応時は deterministic な fallback を残す
- unsupported target を in-place 変更しない

## Projection constraints

- canonical contract から deterministic に生成する
- interface固有の差分は projection 層で吸収する
- `mcp/plugins/skills/workflows/hooks` は reverse sync 自動適用しない

## Repair constraints

- repair は managed 範囲のみ変更する
- append/include target は idempotent を維持する
- user-managed content を保存する

## Status

Extension バックログ実装における基準仕様です。
