# Legacy Governance and Promotion Rule

## Legacy Status

- reference_only
- partially_mapped
- canonicalized

## Promotion Conditions

All required:

- ownership known
- freshness acceptable
- canonical mapping exists

Metadata unknown assets are referenceable but cannot auto-promote.

## Drift Conditions

- canonical conflict
- stale timestamp or stale review
- missing mapped target
- duplicate guidance risk
- deprecated but still referenced

Drift cannot be silently ignored.
Legacy conflict cannot end as warn-only in all cases; strict escalation path is required.

## Resolver API

`resolve_legacy_status(asset) -> legacy_status_decision`
