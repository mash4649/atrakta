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

- instruction surface: canonical instruction intent comes from contract + prompt policy only.
- approval surface: required permission and approval trigger must be equivalent per task type.
- output surface: required labels/format constraints (for example goal-prefix policy) must be enforced consistently.
- execution surface: detect -> plan -> apply and non-interactive behavior must stay consistent.
- quality surface: quick/heavy checks and gate behavior must be consistent.
- safety surface: fail-closed and read-only restrictions must be equivalent.
- routing surface: interface resolution order and fallback behavior must be deterministic.

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

## Compatibility Constraints

- Keep `Fast Path First, Strict Path On Demand`.
- Keep `latest-only` update principle for managed artifacts.
- Keep single-writer determinism for apply/projection side effects.
- Do not introduce automatic repair into the normal `start` critical path.

## Acceptance Criteria

- surfaces above are machine-checkable
- parity drift can be diagnosed and reported by interface
- rules are compatible with fail-closed behavior
- rules do not conflict with fast path, strict path, or latest-only policy

## Non-goals

- forcing byte-identical native file formats across all tools
- replacing tool-native UX with a single generic UX

## Status

Normative baseline for parity backlog implementation.
