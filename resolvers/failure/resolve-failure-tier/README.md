# resolve_failure_tier

Signature:

`resolve_failure_tier(failure_class, context) -> tier_decision`

Implemented in:

- `resolver.go` as `ResolveFailureTier(failureClass string, ctx Context) common.ResolverOutput`

Rules:

- class defaults define `default_tier`, `can_override`, `requires_human_review`
- strict triggers escalate to strict transition
- diagnostics projection failure is handled separately from execution failure
- execution stop and projection stop are modeled independently
- output follows `schemas/operations/inspect-output.schema.json`
