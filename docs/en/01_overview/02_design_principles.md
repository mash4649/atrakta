# Design Principles

[English](./02_design_principles.md) | [日本語](../../ja/01_全体/02_設計原則.md)

## Unified Principle

- Fast Path First, Strict Path On Demand
- Run the fastest safe path by default, then auto-escalate to strict path only when ambiguity/error/threshold breach appears.

## Implementation Rules

- Keep runtime light:
  - runtime metrics are record-only (low overhead)
  - normal events use group commit + phase-end flush
- Keep strict checks in verification:
  - CI fail-closed checks for p95/p99, token budget, and fast-path hit rate
  - continuous fault injection tests (crash-equivalent, lock contention, interruption)
- Auto-escalate when needed:
  - wrapper uses lightweight stamp first; deep check only on suspicion
  - start prefers snapshot fast gate; strict path only on interval/config drift/managed drift
  - `BLOCKED` and `FAIL` events are urgent and synced immediately
- Token efficiency:
  - conventions loading is fixed to staged mode (`index -> relevant sections -> full only when needed`)
- Keep public config minimal:
  - expose only required flags; fine-grained tuning is internally managed via `.atrakta/runtime/meta.v2.json`

## Objective Completion Criteria

- No perceived runtime performance regression
- CI auto-detects performance/safety/integrity regressions
- No degradation in 24h/72h soak runs
- No increase in user operation count
