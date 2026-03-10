# Parity Contract

[English](./06_parity_contract.md) | [日本語](../../ja/02_仕様/06_Parity Contract.md)

## Purpose

Define what "same development experience" means across supported tools.

## Canonical Sources of Truth

- `.atrakta/contract.json`
- `.atrakta/state.json`
- `.atrakta/events.jsonl`

Tool-specific projections are derived artifacts and must not become the source of truth.

## Parity Surfaces

- instruction surface
- approval surface
- output surface
- execution surface
- quality surface
- safety surface
- routing surface

## Projection Rules

- deterministic render from canonical contract
- stable order and formatting for hashing/replay
- unsupported fields must be visible as warnings

## Drift Detection Rules

- compare source hash and render hash
- detect missing projection outputs
- detect managed-block corruption where applicable

## Reverse Sync Policy

- reverse sync is proposal-only
- protected fields are never auto-written back
- apply requires explicit approval

## Acceptance Criteria

- surfaces above are machine-checkable
- parity drift can be diagnosed and reported by interface
- rules are compatible with fail-closed behavior

## Non-goals

- forcing byte-identical native file formats across all tools
- replacing tool-native UX with a single generic UX

## Status

Draft specification for parity backlog implementation.
