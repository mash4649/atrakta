# Parity Contract

[English](../../en/02_spec/06_parity_contract.md) | [日本語](./06_Parity Contract.md)

## Purpose

対応ツールが異なっても「ほぼ同じ開発体験」を成立させるための基準を定義します。

## Canonical sources of truth

- `.atrakta/contract.json`
- `.atrakta/state.json`
- `.atrakta/events.jsonl`

tool固有の設定ファイルは projection 結果であり、正本にはしません。

## Parity surfaces

- instruction surface: canonical な指示意図は contract + prompt policy のみを正本とする。
- approval surface: required_permission と承認判定条件を task 種別ごとに一致させる。
- output surface: goal-prefix などの出力制約を interface 間で同等に適用する。
- execution surface: detect -> plan -> apply と非対話時の挙動を一致させる。
- quality surface: quick/heavy checks と gate 判定を一致させる。
- safety surface: fail-closed / read-only 制約を同等に適用する。
- routing surface: interface 解決順序と fallback を deterministic にする。

## Projection rules

- canonical contract から deterministic に生成する
- 出力順・整形・改行を安定化し hash 比較可能にする
- unsupported は warning として可視化する

## Drift detection rules

- source hash / render hash の差分検知
- projection 欠損検知
- managed block 破損検知

## Reverse sync policy

- reverse sync は proposal-only
- protected field は自動反映しない
- apply は明示承認必須

## Compatibility constraints

- `Fast Path First, Strict Path On Demand` を維持する。
- managed artifact は `latest-only` 方針を維持する。
- apply/projection の副作用は single-writer determinism を維持する。
- 通常 `start` のクリティカルパスへ自動 repair を組み込まない。

## Acceptance criteria

- 上記 surface が機械可読で検証できる
- parity drift を interface 単位で診断できる
- fail-closed 方針と矛盾しない
- fast path / strict path / latest-only 方針と矛盾しない

## Non-goals

- 全ツールの native ファイルをバイト単位で同一化すること
- ツール固有UXを無視して単一UXへ置換すること

## Status

Parity バックログ実装における基準仕様です。
