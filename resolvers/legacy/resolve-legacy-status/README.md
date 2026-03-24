# resolve_legacy_status

Signature:

`resolve_legacy_status(asset) -> legacy_status_decision`

Implemented in:

- `resolver.go` as `ResolveLegacyStatus(Asset) common.ResolverOutput`

Rules:

- statuses: reference_only / partially_mapped / canonicalized
- promotion requires ownership known + freshness acceptable + canonical mapping exists
- unknown metadata blocks auto-promotion
- drift is explicit and never silently ignored
- canonical conflict and missing mapped target trigger strict escalation
