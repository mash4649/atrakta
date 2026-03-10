# Extension Surface

[English](../../en/02_spec/07_extension_surface.md) | [日本語](./07_Extension Surface.md)

## Purpose

拡張要素を contract 上の一級概念として定義します。

## Extension entities

- MCP
- plugin
- skill
- workflow
- hook

## Merge strategy

- 既定: append-first
- 明示モード: `append`, `include`, `replace`

## Brownfield modes

- append mode: 既存ファイルへ managed block を追記
- include mode: include 用ファイルを生成して参照させる

## Unsupported policy

- silent ignore しない
- field path と理由を warning で表示する
- native projection 非対応時は deterministic な fallback を残す

## Projection constraints

- canonical contract から deterministic に生成する
- interface固有の差分は projection 層で吸収する

## Status

Extension バックログ実装向けの draft 仕様です。
