# v0 Execution Plan

## Objective

Build a minimal but implementable contract baseline for Atrakta rewrite.
The baseline must support brownfield and new project onboarding without manual setup.
The operational entrypoint is `atrakta run`.
This rewrite does not require `v0.14.1` command/data compatibility.

## Non-Negotiable Principles

1. Detect -> Plan -> Apply discipline
2. Managed-only destructive mutation
3. Never prune under ambiguity
4. Append-only event observability
5. Deterministic replay and explicit approval

## Delivery Shape

- Contract artifacts are first-class and versioned.
- Resolver IO is explicit and inspectable.
- Mutation is always proposal-gated.
- Onboarding uses `detect -> propose -> accept`.
- Managed runtime writes use `detect -> plan -> apply`.
- Canonical write paths are isolated from projections.

## Versioning Policy

- Rewrite line starts at `1.0.0-alpha.2`.
- Breaking contract changes increment major during alpha.
- Schema additions that preserve compatibility increment minor.
- Documentation-only updates increment patch.

## Issue Order

1. Issue 1: Layer Boundary Contract
2. Issue 2: Guidance Strength and Precedence
3. Issue 3: Projection Model and One-Way Rule
4. Issue 4: Failure Routing and Strict State Machine
5. Issue 5: Managed Scope and Mutation Policy
6. Issue 6: Legacy Governance and Promotion Rule
7. Issue 7: Operations Capability Model
8. Issue 8: Extension Boundary and Evaluation Order
9. Issue 9: Audit Integrity and Retention Policy
10. Issue 10: Inspect Preview and Simulate Interfaces

## Zero-Config Onboarding Flow

### Phase A: Inspect

- Detect project root and asset surfaces.
- Enumerate tools, workflows, instructions, and risk candidates.
- No writes.

### Phase B: Classify

Run resolvers in fixed order:

1. `classify_layer`
2. `resolve_guidance_precedence`
3. `check_projection_eligibility`
4. `resolve_failure_tier`
5. `check_mutation_scope`
6. `resolve_legacy_status`
7. `resolve_extension_order`
8. `resolve_audit_requirements`

### Phase C: Propose

- Build onboarding proposal bundle.
- Keep all write operations as proposal artifacts.

### Phase D: Accept

- Persist only accepted proposals into canonical and generated stores.

## Initial Output Contract

All resolvers must return:

- `input`
- `decision`
- `reason`
- `evidence`
- `next_allowed_action`

## Initial Backlog

1. Minimal schemas for core and canonical entities.
2. Resolver stubs with deterministic IO contract.
3. Contract tests and fixtures.
4. Onboarding proposal bundle schema.
5. Safe default policy profile.

## Done Criteria for v0 Baseline

- Schema ownership boundaries are documented.
- All required resolver APIs exist with explicit IO contract.
- Proposal-only mutation path is available.
- Brownfield-safe defaults are defined.
- Deterministic test fixtures execute for contract and resolver cases.
- Concept coverage for the refresh scope is documented and verified by run-focused E2E.
