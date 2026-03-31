# Projection Model and One-Way Rule

## Projection Eligibility

- allowed: `canonical_policy`, `repo_docs`, `skill_asset`, `workflow_binding`
- conditional: Decision, Result
- forbidden: Task State, Audit Event

Conditional sources cannot be decision root by themselves.
Projection must not depend on runtime temporary state.

## Projection Types

- `durable`
- `ephemeral`
- `diagnostics`

## Evaluation Order

1. canonical first
2. overlay next
3. include next
4. projection render last

## One-Way Rule

Overlay and projection outputs cannot auto-write back to canonical store.
Reverse sync is proposal-only.

## Resolver API

`check_projection_eligibility(source) -> allowed | conditional | forbidden`
