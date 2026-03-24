# resolve_audit_requirements

Signature:

`resolve_audit_requirements(action) -> audit_requirements_decision`

Implemented in:

- `resolver.go` as `ResolveAuditRequirements(Input) common.ResolverOutput`

Rules:

- integrity levels A0/A1/A2/A3 are logical storage-independent contract
- append-only is mandatory
- archival is dry-run first
- destructive cleanup is proposal-only
- audit guarantee shortfall is strict-trigger compatible
