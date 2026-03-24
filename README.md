# Atrakta Canonical Model v0

Version: `1.0.0-alpha.2`

This repository is the new rewrite baseline for Atrakta.
The design is contract-first and resolver-first.
The machine-executable contract lives at `.atrakta/contract.json`.
This line is a refresh track, not a `v0.14.1` compatibility track.

## Design Goals

- Maximum speed with deterministic behavior
- High reliability and high safety by default
- Minimal initial footprint with progressive extension
- Zero-config onboarding through detect -> propose -> accept
- No context drift across surfaces

## Five Pillars

- `schemas/`: contract source of truth
- `canonical/`: single write source for canonical state
- `resolvers/`: deterministic decision logic
- `adapters/`: external binding and projection surfaces
- `tests/`: contract and resolver verification

## First Delivery Scope

1. Define layer boundary and ownership.
2. Define guidance strength and precedence.
3. Define one-way projection model.
4. Define failure routing and strict lifecycle.
5. Define managed scope and mutation policy.
6. Define legacy governance and promotion.
7. Define operations capability model.
8. Define extension boundary and evaluation order.
9. Define audit integrity and retention policy.
10. Define inspect/preview/simulate output contract.

## Startup Rule

- Detect first
- Safe default
- Proposal-first mutation
- Lazy canonicalization
- Progressive disclosure

## Next Step

Start from `docs/architecture/run-contract.md`, then execute
`docs/plan/v0-execution-plan.md` in issue order.

## Quick Start

- `go run ./cmd/atrakta run --project-root . --json`
- `go run ./cmd/atrakta run --project-root . --non-interactive --json`
- `go run ./cmd/atrakta run --project-root . --apply --approve --json`

## Refresh Positioning

- Operational interface is `run` first.
- Legacy command/data parity is intentionally out of scope.
- Evaluation is based on concept coverage, deterministic replay, and safety invariants.

## Documentation Entry Point

- [docs/README.md](docs/README.md)
- Japanese / 日本語: [README.ja.md](README.ja.md), [docs/ja/README.md](docs/ja/README.md)
