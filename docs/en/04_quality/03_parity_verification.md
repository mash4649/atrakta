# 03. Parity Verification

[English](./03_parity_verification.md) | [日本語](../../ja/04_品質/03_Parity検証.md)

## Scope

Parity verification focuses on cross-interface consistency of contract projection and runtime behavior.

## Current Baseline

Use existing verification loops as baseline quality checks:

- `./scripts/dev/verify_loop.sh`
- `./scripts/dev/verify_perf_gate.sh`

## Planned Additions

The parity backlog tracks the following planned checks:

- projection drift detection
- render hash mismatch detection
- managed block corruption detection
- parity diagnostics report

## Output Requirement

Verification must distinguish blocking issues and warnings, with machine-readable output for automation.
