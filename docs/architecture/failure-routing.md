# Failure Routing

## Failure Classes

- policy_failure
- approval_failure
- capability_resolution_failure
- projection_failure
- adapter_execution_failure
- provenance_failure
- audit_integrity_failure
- legacy_conflict_failure
- surface_portability_failure

## Tier Types

- BLOCK
- DEGRADE_TO_STRICT
- PROPOSAL_ONLY
- WARN_ONLY

## Required Mapping

Each failure class defines:

- default_tier
- can_override
- requires_human_review

Fail-closed is not equal to always block.
Execution stop and projection stop are separate controls.
Diagnostics projection failures and execution failures are routed separately.

## Resolver API

`resolve_failure_tier(failure_class, context) -> tier_decision`

## Onboarding Linkage

Zero-config onboarding maps detected conflicts to strict triggers and routes
them through `resolve_failure_tier` as `inferred_failure_routing`.
